# agent-switch — Enhancement Plan (ToolProvider Self-Registration)

> **Goal:** Implement `mcp.ToolProvider` so the module self-registers its MCP tools
> via `srv.RegisterProvider(m)`. Move schema migration into `New(db)`.
>
> **Depends on:** `tinywasm/mcp` `RegisterProvider` + fixed `ToolExecutor` (see `tinywasm/mcp/docs/PLAN.md`).
> **Status:** Pending execution

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Build Tag:** All backend files must use `//go:build !wasm`.
- **No log injection:** The module receives only `db *orm.DB`. No log parameter in `New()`.

---

## Step 1 — Move Schema Migration into `New(db)`

**Target File:** `mcp.go`

`agentswitch.New(db)` must call `db.CreateTable(&AgentSwitch{})` before returning.

```go
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
```

---

## Step 2 — Implement `ToolProvider`

**Target File:** `mcp.go`

Add `GetMCPToolsMetadata()` to make `*Module` implement `mcp.ToolProvider`.
The `Execute` field points directly to the existing handler methods — no adapter needed.

```go
func (m *Module) GetMCPToolsMetadata() []mcp.ToolMetadata {
    return []mcp.ToolMetadata{
        {
            Name:        "get_agent_status",
            Description: "Returns the current agent enabled/disabled status.",
            Execute:     m.GetStatus,
            // No parameters.
        },
        {
            Name:        "toggle_agent_status",
            Description: "Enables or disables the agent. Append-only audit log.",
            Parameters: []mcp.ParameterMetadata{
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
```

---

## Step 3 — Add `RegisterTools`

**Target File:** `mcp.go`

```go
// RegisterTools registers all agent-switch MCP tools on the given server.
// Call once during application startup after New(db).
func (m *Module) RegisterTools(srv *mcp.MCPServer) {
    srv.RegisterProvider(m)
}
```

---

## Step 4 — Update Tests

- Update `mcp_test.go` to verify `GetMCPToolsMetadata()` returns the expected 2 tool names
  with their parameter schemas.
- Add a test that `New(db)` creates the `agent_switch` table in a test DB.
- Run `gotest` — 100% pass required.

---

## Step 5 — Verify & Submit

1. Run `gotest` from project root.
2. Run `gopush 'feat: ToolProvider self-registration, migrate schema in New()'`
