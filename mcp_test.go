//go:build !wasm

package agentswitch

import (
	"context"
	"strings"
	"testing"

	"github.com/tinywasm/sqlite"
)

func setupTestModule(t *testing.T) *Module {
	db, _ := sqlite.Open(":memory:")
	m, _ := New(db)
	return m
}

func TestGetMCPToolsMetadata(t *testing.T) {
	m := setupTestModule(t)
	tools := m.GetMCPTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "get_agent_status" {
		t.Errorf("expected tool 0 to be get_agent_status, got %s", tools[0].Name)
	}

	if tools[1].Name != "toggle_agent_status" {
		t.Errorf("expected tool 1 to be toggle_agent_status, got %s", tools[1].Name)
	}

	if len(tools[1].Parameters) != 3 {
		t.Errorf("expected 3 parameters for toggle_agent_status, got %d", len(tools[1].Parameters))
	}
}

func TestGetAgentStatus_Enabled(t *testing.T) {
	m := setupTestModule(t)
	m.db.Create(&AgentSwitch{
		ID:        m.uid.GetNewID(),
		IsEnabled: true,
		ChangedBy: "u1",
		Reason:    "test",
	})

	res, err := m.GetStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mRes, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", res)
	}

	if mRes["is_enabled"] != true {
		t.Errorf("expected is_enabled to be true, got %v", mRes["is_enabled"])
	}
	if mRes["changed_by"] != "u1" {
		t.Errorf("expected changed_by to be 'u1', got %v", mRes["changed_by"])
	}
}

func TestGetAgentStatus_NoHistory(t *testing.T) {
	m := setupTestModule(t)

	res, err := m.GetStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mRes, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", res)
	}

	if mRes["is_enabled"] != false {
		t.Errorf("expected is_enabled to be false, got %v", mRes["is_enabled"])
	}
	if mRes["changed_at"] != int64(0) {
		t.Errorf("expected changed_at to be 0, got %v", mRes["changed_at"])
	}
}

func TestGetAgentStatus_ReturnsLatestOnly(t *testing.T) {
	m := setupTestModule(t)
	m.db.Create(&AgentSwitch{
		ID:        m.uid.GetNewID(),
		IsEnabled: true,
		ChangedBy: "u1",
		Reason:    "1",
	})
	m.db.Create(&AgentSwitch{
		ID:        m.uid.GetNewID(),
		IsEnabled: true,
		ChangedBy: "u2",
		Reason:    "2",
	})
	m.db.Create(&AgentSwitch{
		ID:        m.uid.GetNewID(),
		IsEnabled: false,
		ChangedBy: "u3",
		Reason:    "3",
	})

	res, err := m.GetStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mRes, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", res)
	}

	if mRes["is_enabled"] != false {
		t.Errorf("expected is_enabled to be false, got %v", mRes["is_enabled"])
	}
	if mRes["changed_by"] != "u3" {
		t.Errorf("expected changed_by to be 'u3', got %v", mRes["changed_by"])
	}
}

func TestToggleAgentStatus_Enable(t *testing.T) {
	m := setupTestModule(t)

	res, err := m.Toggle(context.Background(), map[string]any{
		"is_enabled": true,
		"changed_by": "u1",
		"reason":     "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mRes, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", res)
	}

	if mRes["ok"] != true {
		t.Errorf("expected ok to be true, got %v", mRes["ok"])
	}
	if mRes["is_enabled"] != true {
		t.Errorf("expected is_enabled to be true, got %v", mRes["is_enabled"])
	}

	rows, err := ReadAllAgentSwitch(m.db.Query(&AgentSwitch{}))
	if err != nil {
		t.Fatalf("unexpected error querying db: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row in DB, got %d", len(rows))
	}
	if rows[0].IsEnabled != true {
		t.Errorf("expected DB row to be enabled")
	}
	if rows[0].ChangedBy != "u1" {
		t.Errorf("expected DB row changed_by to be u1, got %v", rows[0].ChangedBy)
	}
}

func TestToggleAgentStatus_MissingIsEnabled(t *testing.T) {
	m := setupTestModule(t)

	_, err := m.Toggle(context.Background(), map[string]any{
		"changed_by": "u1",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "params invalid") && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected params invalid error, got %v", err)
	}
}

func TestToggleAgentStatus_MissingChangedBy(t *testing.T) {
	m := setupTestModule(t)

	_, err := m.Toggle(context.Background(), map[string]any{
		"is_enabled": true,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "params invalid") && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected params invalid error, got %v", err)
	}
}

func TestToggleAgentStatus_AppendOnly(t *testing.T) {
	m := setupTestModule(t)

	_, err := m.Toggle(context.Background(), map[string]any{
		"is_enabled": true,
		"changed_by": "u1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = m.Toggle(context.Background(), map[string]any{
		"is_enabled": false,
		"changed_by": "u2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, err := ReadAllAgentSwitch(m.db.Query(&AgentSwitch{}))
	if err != nil {
		t.Fatalf("unexpected error querying db: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows in DB, got %d", len(rows))
	}
}
