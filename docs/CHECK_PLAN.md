# agent-switch — Implementation Plan.

> **Module:** `github.com/veltylabs/agent-switch`
> **Package:** `agentswitch`
> **Goal:** Implement `GetStatus` and `Toggle` handlers. Append-only audit log — no UPDATE or DELETE permitted. New table auto-created via ORM on startup.
>
> This module is a standalone Lego piece. It has no knowledge of which application will use it.

---

## Development Rules

- **SRP:** Every file has a single, well-defined purpose.
- **DI:** DB injected via `*orm.DB`. No global state.
- **Flat structure:** All files in repo root — no subdirectories.
- **Max 500 lines per file.**
- **Testing:** `gotest` (not `go test`). Mock all external interfaces. DDT.
- **ORM:** `tinywasm/orm` + `ormc` code generator. Run `ormc` from repo root.
- **Time:** Use `github.com/tinywasm/time`. NEVER use standard `time` package.
- **Errors:** `tinywasm/fmt` only — Noun+Adjective word order.

### Installation Prerequisites

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
go install github.com/tinywasm/orm/cmd/ormc@latest
```

---

## go.mod Dependencies

```bash
go get github.com/tinywasm/orm@latest
go get github.com/tinywasm/fmt@latest
go get github.com/tinywasm/unixid@latest
go get github.com/tinywasm/sqlite@latest
```

---

## Append-Only Design

The `agent_switch` table acts as an audit log:
- Each row records an enable/disable event.
- The **current status** is the value of the LAST row (`ORDER BY id DESC LIMIT 1`).
- There are NO updates to existing rows. Disabling = INSERT a new row with `is_enabled=false`.

---

## Database Schema

```sql
CREATE TABLE agent_switch (
    id          VARCHAR(255) PRIMARY KEY,  -- string PK via unixid
    is_enabled  BOOLEAN      NOT NULL,
    changed_by  VARCHAR(255) NOT NULL,     -- actor identifier (injected by caller)
    reason      TEXT                       -- optional reason
);
```

---

## Handler Contracts

### `GetStatus`

Input: *(none)*

Output:
```json
{
  "is_enabled": true,
  "changed_by": "user_01jk...",
  "changed_at": 1710000000,
  "reason": "manual activation"
}
```

Returns most recent row. If no rows: `{ "is_enabled": false, "changed_at": 0 }` (safe default).

### `Toggle`

Input:
```json
{
  "is_enabled": false,
  "changed_by": "user_01jk...",
  "reason": "maintenance window"
}
```

Output:
```json
{ "ok": true, "is_enabled": false }
```

Validation:
- `is_enabled` (bool) — required
- `changed_by` (string) — required, non-empty; the **caller** is responsible for injecting the actor identity (e.g. from JWT claims) before calling this handler or passing it in args.

---

## Files to Create

| File | Action | Purpose |
|---|---|---|
| `model.go` | Create | `AgentSwitch` struct (append-only, string PK via unixid) |
| `model_orm.go` | Generate | Run `ormc` |
| `mcp.go` | Create | `Module` struct + exported handler methods |
| `mcp_test.go` | Create | Black-box tests with mock DB |

---

## Step 1 — Model (`model.go`)

```go
//go:build !wasm

package agentswitch

import "github.com/tinywasm/orm"

// AgentSwitch records a single agent enable/disable event.
// Append-only: INSERT only. No UPDATE. No DELETE.
type AgentSwitch struct {
    ID        string `db:"pk"`       // set by caller via unixid before db.Create()
    IsEnabled bool   `db:"not_null"`
    ChangedBy string `db:"not_null"` // actor identity injected by the application layer
    Reason    string                 // optional free-text reason
}

func (a *AgentSwitch) TableName() string { return "agent_switch" }
```

**Generate ORM code:**
```bash
# From repo root:
ormc
# Creates: model_orm.go
```

---

## Step 2 — Module (`mcp.go`)

```go
//go:build !wasm

package agentswitch

import (
    "context"

    "github.com/tinywasm/fmt"
    "github.com/tinywasm/orm"
    "github.com/tinywasm/unixid"
)

type Module struct {
    db  *orm.DB
    uid *unixid.UnixID
}

func New(db *orm.DB) (*Module, error) {
    u, err := unixid.NewUnixID()
    if err != nil {
        return nil, err
    }
    return &Module{db: db, uid: u}, nil
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
```

> **Note on auth:** `Toggle` requires `changed_by` in args but does NOT check JWT itself. The application layer (e.g. mjosefa-cms) is responsible for extracting `changed_by` from the JWT claims and injecting it into args, or wrapping this handler with an auth guard.

---

## Step 3 — Tests (`mcp_test.go`)

Integration tests using `github.com/tinywasm/sqlite` with an in-memory database for real ORM behavior.

```go
func setupTestModule(t *testing.T) *Module {
    db, _ := sqlite.Open(":memory:")
    db.CreateTable(&AgentSwitch{})
    m, _ := New(db)
    return m
}
```

```
TestGetAgentStatus_Enabled
  - Seed 1 row with is_enabled=true
  - Assert response has is_enabled=true, changed_by populated

TestGetAgentStatus_NoHistory
  - Start with empty table
  - Assert response has is_enabled=false, changed_at=0 (safe default)

TestGetAgentStatus_ReturnsLatestOnly
  - Insert 3 rows; latest is_enabled=false
  - Assert response reflects false (most recent)

TestToggleAgentStatus_Enable
  - args = {"is_enabled": true, "changed_by": "u1", "reason": "test"}
  - Call Toggle
  - Verify 1 row exists in DB with is_enabled=true, changed_by="u1"
  - Assert response {"ok": true, "is_enabled": true}

TestToggleAgentStatus_MissingIsEnabled
  - args = {} (no is_enabled key)
  - Assert error contains "invalid params"

TestToggleAgentStatus_MissingChangedBy
  - args = {"is_enabled": true} (no changed_by)
  - Assert error contains "invalid params"

TestToggleAgentStatus_AppendOnly
  - Call Toggle twice
  - Verify DB has exactly 2 rows
  - Assert no updates occurred
```

```bash
gotest -run TestGetAgentStatus
gotest -run TestToggleAgentStatus
```

---

## Checklist

- [ ] `model.go` — `AgentSwitch` struct with string PK; `TableName()` returns `"agent_switch"`
- [ ] `ormc` run from repo root — `model_orm.go` generated (contains `ReadAllAgentSwitch`, `AgentSwitchMeta`)
- [ ] `mcp.go` — NO import of `mjosefa-cms` or any application package
- [ ] `mcp.go` — `GetStatus` and `Toggle` are exported, match signature `func(context.Context, map[string]any) (any, error)`
- [ ] `mcp.go` — `Toggle` uses `db.Create()` ONLY (no `Update`/`Delete`)
- [ ] All 7 test cases pass: `gotest -run TestGetAgentStatus` and `gotest -run TestToggleAgentStatus`
- [ ] `go build ./...` succeeds
- [ ] `gopush 'implement GetStatus and Toggle handlers'`
