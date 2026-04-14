# agent-switch
<img src="docs/img/badges.svg">

Append-only runtime switch for AI agent enable/disable. Every state change is persisted
as a new row; the latest row represents the current state.

## MCP Tools

| Tool | Description |
|------|-------------|
| `get_agent_status` | Returns current enabled state, actor, timestamp, and reason. |
| `toggle_agent_status` | Inserts a new audit row (enable or disable). |

## Quick Start

```go
import agentswitch "github.com/veltylabs/agent-switch"

m, err := agentswitch.New(db)   // creates table + initialises module
m.RegisterTools(srv)            // registers MCP tools on *mcp.MCPServer
```

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Domain scope, patterns, and MCP tool reference |
| [Database Diagram](docs/diagrams/database.md) | Schema diagram |
| [SKILL.md](docs/SKILL.md) | LLM-friendly condensed summary |
