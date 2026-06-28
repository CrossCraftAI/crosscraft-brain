package core

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, imapNode) }

// imapNode reads emails from an IMAP or POP3 server.
//
// NOTE: This implementation provides basic IMAP and POP3 support via stdlib
// net/textproto. For full IMAP support (IDLE, SEARCH, FLAGS), add
// github.com/emersion/go-imap.
var imapNode = schema.NodeDefinition{
	Type: "core.readEmail", Label: "Read Email (IMAP/POP3)", Group: "integration", Icon: "Mail",
	Description: "Read and list emails from an IMAP or POP3 server. Use a credential of type 'imapAuth' with keys: host, port, user, password, tls.",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "action", Label: "Action", Type: "select", Default: "list", Options: []schema.ParamOption{
			{Label: "List Emails", Value: "list"},
			{Label: "Read Email", Value: "read"},
		}},
		{Name: "protocol", Label: "Protocol", Type: "select", Default: "imap", Options: []schema.ParamOption{
			{Label: "IMAP", Value: "imap"},
			{Label: "POP3", Value: "pop3"},
		}},
		{Name: "host", Label: "Server host", Type: "string", Required: true, Placeholder: "imap.gmail.com"},
		{Name: "port", Label: "Port", Type: "number", Default: float64(993)},
		{Name: "user", Label: "Username / Email", Type: "string", Required: true},
		{Name: "password", Label: "Password / App password", Type: "string", Required: true},
		{Name: "useTLS", Label: "Use TLS", Type: "boolean", Default: true},
		{Name: "mailbox", Label: "Mailbox", Type: "string", Default: "INBOX",
			ShowWhen: &schema.ShowWhen{Param: "protocol", Equals: []any{"imap"}}},
		{Name: "maxResults", Label: "Max results", Type: "number", Default: float64(25)},
		{Name: "messageId", Label: "Message ID (sequence number)", Type: "string",
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"read"}}},
		{Name: "filter", Label: "Search filter", Type: "string", Placeholder: "UNSEEN",
			ShowWhen: &schema.ShowWhen{Param: "protocol", Equals: []any{"imap"}}},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		action := asString(ctx.Params["action"], "list")
		protocol := asString(ctx.Params["protocol"], "imap")
		host := asString(ctx.Params["host"], "")
		port := int(asFloat(ctx.Params["port"], 993))
		user := asString(ctx.Params["user"], "")
		password := asString(ctx.Params["password"], "")
		useTLS := true
		if v, ok := ctx.Params["useTLS"]; ok {
			useTLS = isTruthy(v)
		}
		mailbox := asString(ctx.Params["mailbox"], "INBOX")
		maxResults := int(asFloat(ctx.Params["maxResults"], 25))
		messageID := asString(ctx.Params["messageId"], "")
		filter := asString(ctx.Params["filter"], "")

		if host == "" || user == "" || password == "" {
			return schema.NodeResult{}, fmt.Errorf("email: host, user, and password are required")
		}

		addr := net.JoinHostPort(host, strconv.Itoa(port))
		var conn net.Conn
		var err error

		if useTLS {
			dialer := &net.Dialer{Timeout: 15 * time.Second}
			conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
				ServerName: host,
			})
		} else {
			conn, err = net.DialTimeout("tcp", addr, 15*time.Second)
		}
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("email: connect: %w", err)
		}
		defer conn.Close()

		switch protocol {
		case "pop3":
			return handlePOP3(conn, action, user, password, messageID, maxResults)
		default:
			return handleIMAP(conn, action, user, password, mailbox, filter, messageID, maxResults)
		}
	},
}

// ---------------------------------------------------------------------------
// IMAP client (basic)
// ---------------------------------------------------------------------------

func handleIMAP(conn net.Conn, action, user, password, mailbox, filter, messageID string, maxResults int) (schema.NodeResult, error) {
	tp := textproto.NewConn(conn)

	// Read greeting
	_, err := tp.ReadLine()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("imap: greeting: %w", err)
	}

	sendIMAP := func(cmd string, args ...string) (string, error) {
		full := cmd
		if len(args) > 0 {
			full += " " + strings.Join(args, " ")
		}
		if err := tp.PrintfLine("%s", full); err != nil {
			return "", fmt.Errorf("imap: send %q: %w", cmd, err)
		}
		// Read response lines until the tagged response
		var lines []string
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return "", fmt.Errorf("imap: read: %w", err)
			}
			lines = append(lines, line)
			if strings.HasPrefix(line, "A001 ") || strings.HasPrefix(line, "A002 ") ||
				strings.HasPrefix(line, "A003 ") || strings.HasPrefix(line, "A004 ") {
				break
			}
		}
		return strings.Join(lines, "\n"), nil
	}

	// Login
	resp, err := sendIMAP("A001 LOGIN", user, password)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("imap login: %w", err)
	}
	if strings.Contains(resp, "A001 NO") || strings.Contains(resp, "A001 BAD") {
		return schema.NodeResult{}, fmt.Errorf("imap: login failed")
	}

	// Select mailbox
	if _, err := sendIMAP("A002 SELECT", mailbox); err != nil {
		return schema.NodeResult{}, fmt.Errorf("imap select: %w", err)
	}

	switch action {
	case "read":
		if messageID == "" {
			return schema.NodeResult{}, fmt.Errorf("imap read: messageId is required")
		}
		resp, err := sendIMAP("A003 FETCH", messageID, "BODY[]")
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("imap fetch: %w", err)
		}
		email := parseIMAPFetchResult(resp)
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {email}}}, nil

	default: // list
		searchCmd := "1:" + strconv.Itoa(maxResults)
		if filter != "" {
			searchCmd = filter
		}
		resp, err := sendIMAP("A003 FETCH", searchCmd, "(FLAGS INTERNALDATE BODY[HEADER.FIELDS (FROM SUBJECT DATE)])")
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("imap fetch: %w", err)
		}
		emails := parseIMAPListResult(resp)
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": emails}}, nil
	}
}

func parseIMAPFetchResult(raw string) schema.Item {
	result := map[string]any{
		"raw": raw,
	}
	// Extract basic headers from the raw response
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Subject: ") {
			result["subject"] = strings.TrimPrefix(line, "Subject: ")
		}
		if strings.HasPrefix(line, "From: ") {
			result["from"] = strings.TrimPrefix(line, "From: ")
		}
		if strings.HasPrefix(line, "Date: ") {
			result["date"] = strings.TrimPrefix(line, "Date: ")
		}
		if strings.HasPrefix(line, "To: ") {
			result["to"] = strings.TrimPrefix(line, "To: ")
		}
	}
	return schema.Item{JSON: result}
}

func parseIMAPListResult(raw string) []schema.Item {
	var emails []schema.Item
	current := map[string]any{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* ") && strings.Contains(line, "FETCH") {
			if len(current) > 1 {
				emails = append(emails, schema.Item{JSON: current})
			}
			current = map[string]any{}
			// Extract sequence number
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 2 {
				current["seqNum"] = parts[1]
			}
		}
		if strings.HasPrefix(line, "Subject: ") {
			current["subject"] = strings.TrimPrefix(line, "Subject: ")
		}
		if strings.HasPrefix(line, "From: ") {
			current["from"] = strings.TrimPrefix(line, "From: ")
		}
		if strings.HasPrefix(line, "Date: ") {
			current["date"] = strings.TrimPrefix(line, "Date: ")
		}
	}
	if len(current) > 1 {
		emails = append(emails, schema.Item{JSON: current})
	}
	return emails
}

// ---------------------------------------------------------------------------
// POP3 client
// ---------------------------------------------------------------------------

func handlePOP3(conn net.Conn, action, user, password, messageID string, maxResults int) (schema.NodeResult, error) {
	tp := textproto.NewConn(conn)

	// Read greeting
	_, err := tp.ReadLine()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("pop3: greeting: %w", err)
	}

	sendPOP3 := func(cmd string) (string, error) {
		if err := tp.PrintfLine("%s", cmd); err != nil {
			return "", fmt.Errorf("pop3: send %q: %w", cmd, err)
		}
		// For multi-line responses (LIST, RETR, TOP), read until "."
		if strings.HasPrefix(cmd, "RETR") || strings.HasPrefix(cmd, "LIST") || strings.HasPrefix(cmd, "TOP") {
			var lines []string
			for {
				line, err := tp.ReadLine()
				if err != nil {
					return "", fmt.Errorf("pop3: read: %w", err)
				}
				if line == "." {
					break
				}
				lines = append(lines, line)
			}
			return strings.Join(lines, "\n"), nil
		}
		line, err := tp.ReadLine()
		if err != nil {
			return "", fmt.Errorf("pop3: read: %w", err)
		}
		return line, nil
	}

	// Login
	if _, err := sendPOP3("USER " + user); err != nil {
		return schema.NodeResult{}, fmt.Errorf("pop3 user: %w", err)
	}
	resp, err := sendPOP3("PASS " + password)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("pop3 pass: %w", err)
	}
	if strings.HasPrefix(resp, "-ERR") {
		return schema.NodeResult{}, fmt.Errorf("pop3: login failed: %s", resp)
	}

	switch action {
	case "read":
		if messageID == "" {
			return schema.NodeResult{}, fmt.Errorf("pop3 read: messageId is required")
		}
		resp, err := sendPOP3("RETR " + messageID)
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("pop3 retr: %w", err)
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {parsePOP3Message(resp)}}}, nil

	default: // list
		// Get list of messages
		listResp, err := sendPOP3("LIST")
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("pop3 list: %w", err)
		}
		lines := strings.Split(listResp, "\n")
		var items []schema.Item
		for i, line := range lines {
			if i >= maxResults {
				break
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				items = append(items, schema.Item{JSON: map[string]any{
					"seqNum": parts[0],
					"size":   parts[1],
				}})
			}
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
	}
}

func parsePOP3Message(raw string) schema.Item {
	result := map[string]any{"raw": raw}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Subject: ") {
			result["subject"] = strings.TrimPrefix(line, "Subject: ")
		}
		if strings.HasPrefix(line, "From: ") {
			result["from"] = strings.TrimPrefix(line, "From: ")
		}
		if strings.HasPrefix(line, "Date: ") {
			result["date"] = strings.TrimPrefix(line, "Date: ")
		}
	}
	return schema.Item{JSON: result}
}
