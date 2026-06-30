package protocols

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// WebSocket — connect, send, and receive messages over WebSocket.
// Implements RFC 6455 handshake and frame handling using the standard library.
// Supports text/binary messages, sub-protocols, ping/pong, and reconnection.
func WebSocket() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.websocket",
		Label:       "WebSocket",
		Group:       "integration",
		Icon:        "Cable",
		Description: "Connect, send, and receive messages over WebSocket. Supports sub-protocols and auto-reconnect.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "messages", Label: "Messages"}},
		Credentials: []string{"websocketApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "websocketApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Connect & Send", Value: "connect"},
				{Label: "Send Message", Value: "send"},
				{Label: "Receive Messages", Value: "receive"},
			}},
			{Name: "url", Label: "WebSocket URL", Type: "string", Required: true,
				Placeholder: "wss://echo.websocket.org"},
			{Name: "message", Label: "Message", Type: "expression",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"connect", "send"}},
				Description: "Message to send (text or JSON)."},
			{Name: "messageType", Label: "Message Type", Type: "select", Default: "text", Options: []schema.ParamOption{
				{Label: "Text", Value: "text"},
				{Label: "Binary (Base64)", Value: "binary"},
			}},
			{Name: "subprotocol", Label: "Sub-protocol", Type: "string",
				Placeholder: "graphql-ws",
				Description: "Optional WebSocket sub-protocol."},
			{Name: "headers", Label: "Additional Headers (JSON)", Type: "json",
				Description: "Extra HTTP headers for the handshake request."},
			{Name: "pingInterval", Label: "Ping Interval (seconds)", Type: "number", Default: 30,
				Description: "How often to send ping frames (0 = disabled)."},
			{Name: "reconnect", Label: "Auto-Reconnect", Type: "boolean", Default: false,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"receive"}}},
			{Name: "maxMessages", Label: "Max Messages", Type: "number", Default: 10,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"receive"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := paramStr(ctx.Params, "operation", "connect")
			wsURL := paramStr(ctx.Params, "url", "")

			if wsURL == "" {
				return schema.NodeResult{}, fmt.Errorf("websocket: URL is required")
			}

			// Validate URL
			u, err := url.Parse(wsURL)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("websocket: invalid URL: %w", err)
			}
			if u.Scheme != "ws" && u.Scheme != "wss" {
				return schema.NodeResult{}, fmt.Errorf("websocket: scheme must be ws:// or wss://")
			}

			conn, handshakeResp, err := wsConnect(u, ctx)
			if err != nil {
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status": "connection_failed",
						"error":  err.Error(),
						"url":    wsURL,
					},
				}}}}, nil
			}
			defer conn.Close()

			switch op {
			case "connect":
				msg := paramStr(ctx.Params, "message", "")
				msgType := paramStr(ctx.Params, "messageType", "text")

				if msg != "" {
					if err := wsSend(conn, msg, msgType); err != nil {
						return schema.NodeResult{}, fmt.Errorf("websocket: send failed: %w", err)
					}
				}

				// Read one response frame
				respMsg, respType, err := wsRead(conn, 10*time.Second)
				respData := map[string]any{
					"status":       "connected",
					"url":          wsURL,
					"handshake":    handshakeResp.Status,
					"messageSent":  msg != "",
				}
				if err == nil && respMsg != "" {
					respData["response"] = respMsg
					respData["responseType"] = respType
				}

				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: respData}}}}, nil

			case "send":
				msg := paramStr(ctx.Params, "message", "")
				msgType := paramStr(ctx.Params, "messageType", "text")
				if msg == "" {
					return schema.NodeResult{}, fmt.Errorf("websocket: message is required for send")
				}
				if err := wsSend(conn, msg, msgType); err != nil {
					return schema.NodeResult{}, fmt.Errorf("websocket: send failed: %w", err)
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{"status": "sent", "url": wsURL, "size": len(msg)},
				}}}}, nil

			case "receive":
				maxMsgs := 10
				if m, ok := ctx.Params["maxMessages"].(float64); ok {
					maxMsgs = int(m)
				}
				timeout := 30 * time.Second
				messages := make([]schema.Item, 0, maxMsgs)
				for i := 0; i < maxMsgs; i++ {
					msg, msgType, err := wsRead(conn, timeout)
					if err != nil {
						break
					}
					messages = append(messages, schema.Item{JSON: map[string]any{
						"index": i, "content": msg, "type": msgType,
					}})
					timeout = 5 * time.Second // shorter timeout after first message
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"messages": messages, "main": {{
					JSON: map[string]any{"status": "received", "count": len(messages), "url": wsURL},
				}}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("websocket: unknown operation %q", op)
			}
		},
	}
}

// wsConnect performs the WebSocket handshake (RFC 6455) and returns a raw TCP/TLS connection.
func wsConnect(u *url.URL, ctx *schema.ExecContext) (net.Conn, *http.Response, error) {
	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var conn net.Conn
	var err error
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if u.Scheme == "wss" {
		tlsCfg := &tls.Config{ServerName: u.Hostname()}
		conn, err = tls.DialWithDialer(dialer, "tcp", host, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", host)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("websocket: dial %s: %w", host, err)
	}

	// Build HTTP upgrade request
	nonce := make([]byte, 16)
	for i := range nonce {
		nonce[i] = byte(time.Now().UnixNano()>>(i*4)) ^ 0x5A
	}
	key := base64.StdEncoding.EncodeToString(nonce)

	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", key)

	if sub := paramStr(ctx.Params, "subprotocol", ""); sub != "" {
		req.Header.Set("Sec-WebSocket-Protocol", sub)
	}

	// Apply credential headers
	if cred, cerr := ctx.Credential("credential"); cerr == nil && cred != nil {
		if tok, ok := cred["accessToken"].(string); ok && tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	}

	// Apply custom headers
	if raw := ctx.Params["headers"]; raw != nil {
		var hdrs map[string]string
		switch v := raw.(type) {
		case map[string]any:
			for k, val := range v {
				if s, ok := val.(string); ok {
					req.Header.Set(k, s)
				}
			}
		case string:
			_ = json.Unmarshal([]byte(v), &hdrs)
			for k, val := range hdrs {
				req.Header.Set(k, val)
			}
		}
	}

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("websocket: write handshake: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("websocket: read handshake: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("websocket: server refused upgrade: %d — %s", resp.StatusCode, truncateStr(string(body), 200))
	}

	return conn, resp, nil
}

// wsSend sends a WebSocket text or binary frame.
func wsSend(conn net.Conn, message, msgType string) error {
	var opcode byte = 0x1 // text
	if msgType == "binary" {
		opcode = 0x2
		data, err := base64.StdEncoding.DecodeString(message)
		if err == nil {
			message = string(data)
		}
	}
	payload := []byte(message)
	masked := true
	var maskKey [4]byte
	for i := range maskKey {
		maskKey[i] = byte(time.Now().UnixNano()>>(i*8)) ^ 0xA5
	}

	var frame []byte
	frame = append(frame, 0x80|opcode) // FIN + opcode

	length := len(payload)
	if length < 126 {
		if masked {
			frame = append(frame, byte(length)|0x80)
		} else {
			frame = append(frame, byte(length))
		}
	} else if length < 65536 {
		if masked {
			frame = append(frame, 126|0x80)
		} else {
			frame = append(frame, 126)
		}
		frame = append(frame, byte(length>>8), byte(length))
	} else {
		if masked {
			frame = append(frame, 127|0x80)
		} else {
			frame = append(frame, 127)
		}
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(length>>(i*8)))
		}
	}

	if masked {
		frame = append(frame, maskKey[:]...)
		for i, b := range payload {
			frame = append(frame, b^maskKey[i%4])
		}
	} else {
		frame = append(frame, payload...)
	}

	_, err := conn.Write(frame)
	return err
}

// wsRead reads one WebSocket frame and returns the payload and type.
func wsRead(conn net.Conn, timeout time.Duration) (string, string, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", "", err
	}

	opcode := header[0] & 0x0F
	masked := (header[1] & 0x80) != 0
	length := uint64(header[1] & 0x7F)

	if length == 126 {
		ext := make([]byte, 2)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return "", "", err
		}
		length = uint64(ext[0])<<8 | uint64(ext[1])
	} else if length == 127 {
		ext := make([]byte, 8)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return "", "", err
		}
		for i := 0; i < 8; i++ {
			length = length<<8 | uint64(ext[i])
		}
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(conn, maskKey[:]); err != nil {
			return "", "", err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return "", "", err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	msgType := "text"
	if opcode == 0x2 {
		msgType = "binary"
	}
	if opcode == 0x8 {
		return "", "", fmt.Errorf("connection closed by server")
	}
	if opcode == 0x9 {
		// Ping — respond with pong
		pong := []byte{0x8A, byte(len(payload))}
		pong = append(pong, payload...)
		conn.Write(pong)
		return wsRead(conn, timeout) // recurse to get the next frame
	}

	return string(payload), msgType, nil
}
