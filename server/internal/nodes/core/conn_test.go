package core

import (
	"encoding/base64"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSSHExec(t *testing.T) {
	// This test verifies the node builds and basic param validation works.
	// SSH connection tests require a real server; here we check error paths.
	_, err := sshNode.Execute(ctxFor(nil, map[string]any{
		"action": "exec", "host": "", "user": "", "command": "hostname",
	}))
	if err == nil {
		t.Fatal("expected error for missing host/user")
	}
	if !contains(err.Error(), "host") {
		t.Fatalf("expected host error, got: %v", err)
	}
}

func TestSSHExecParams(t *testing.T) {
	def := sshNode
	if def.Type != "core.ssh" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) < 5 {
		t.Fatalf("expected at least 5 params, got %d", len(def.Params))
	}
}

func TestFTPParams(t *testing.T) {
	def := ftpNode
	if def.Type != "core.ftp" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) < 6 {
		t.Fatalf("expected at least 6 params, got %d", len(def.Params))
	}
}

func TestFTPValidation(t *testing.T) {
	_, err := ftpNode.Execute(ctxFor(nil, map[string]any{
		"action": "list", "host": "", "user": "",
	}))
	if err == nil {
		t.Fatal("expected error for missing host/user")
	}
}

func TestIMAPParams(t *testing.T) {
	def := imapNode
	if def.Type != "core.readEmail" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) < 5 {
		t.Fatalf("expected at least 5 params, got %d", len(def.Params))
	}
}

func TestIMAPValidation(t *testing.T) {
	_, err := imapNode.Execute(ctxFor(nil, map[string]any{
		"action": "list", "host": "", "user": "", "password": "",
	}))
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	if !contains(err.Error(), "host") && !contains(err.Error(), "required") {
		t.Fatalf("expected credential error, got: %v", err)
	}
}

var _ = base64.StdEncoding
var _ = schema.ExecContext{}
