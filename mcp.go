//go:build !wasm

package agentswitch

import (
	"context"

	"github.com/tinywasm/fmt"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/unixid"
)

type Module struct {
	db  *orm.DB
	uid *unixid.UnixID
}

func (m *Module) GetMCPTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "get_agent_status",
			Description: "Returns the current agent enabled/disabled status.",
			Execute:     m.GetStatus,
		},
		{
			Name:        "toggle_agent_status",
			Description: "Enables or disables the agent. Append-only audit log.",
			Parameters: []mcp.Parameter{
				{
					Name:        "is_enabled",
					Description: "true to enable the agent, false to disable.",
					Required:    true,
					Type:        "boolean",
				},
				{
					Name:        "changed_by",
					Description: "ID or name of the user making the change.",
					Required:    true,
					Type:        "string",
				},
				{
					Name:        "reason",
					Description: "Optional reason for the change.",
					Required:    false,
					Type:        "string",
				},
			},
			Execute: m.Toggle,
		},
	}
}

func New(db *orm.DB) (*Module, error) {
	if err := db.CreateTable(&AgentSwitch{}); err != nil {
		return nil, err
	}
	u, err := unixid.NewUnixID()
	if err != nil {
		return nil, err
	}
	return &Module{db: db, uid: u}, nil
}

// RegisterTools registers all agent-switch MCP tools on the given server.
// Call once during application startup after New(db).
func (m *Module) RegisterTools(srv *mcp.MCPServer) {
	srv.RegisterProvider(m)
}

// GetStatus returns the current agent enabled/disabled state.
// Signature matches ToolHandler: func(context.Context, map[string]any) (any, error)
func (m *Module) GetStatus(ctx context.Context, args map[string]any) (any, error) {
	rows, err := ReadAllAgentSwitch(
		m.db.Query(&AgentSwitch{}).OrderBy(AgentSwitchMeta.ID).Desc().Limit(1),
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]any{"is_enabled": false, "changed_at": int64(0)}, nil
	}
	r := rows[0]

	// Extract timestamp from unixid
	ts, _, err := m.uid.Parse(r.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"is_enabled": r.IsEnabled,
		"changed_by": r.ChangedBy,
		"changed_at": ts,
		"reason":     r.Reason,
	}, nil
}

// Toggle inserts a new audit row. Append-only — never updates existing rows.
// Signature matches ToolHandler: func(context.Context, map[string]any) (any, error)
// The caller is responsible for injecting "changed_by" into args (e.g. from JWT claims).
func (m *Module) Toggle(ctx context.Context, args map[string]any) (any, error) {
	isEnabled, ok := args["is_enabled"].(bool)
	if !ok {
		return nil, fmt.Err("params", "invalid") // EN: Params Invalid
	}
	changedBy, _ := args["changed_by"].(string)
	if changedBy == "" {
		return nil, fmt.Err("params", "invalid") // EN: Params Invalid
	}
	reason, _ := args["reason"].(string)

	row := &AgentSwitch{
		ID:        m.uid.GetNewID(),
		IsEnabled: isEnabled,
		ChangedBy: changedBy,
		Reason:    reason,
	}
	if err := m.db.Create(row); err != nil {
		return nil, fmt.Err("database", "unavailable") // EN: Database Unavailable
	}

	return map[string]any{"ok": true, "is_enabled": isEnabled}, nil
}
