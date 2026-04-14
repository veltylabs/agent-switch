//go:build !wasm

package agentswitch

// AgentSwitch records a single agent enable/disable event.
// Append-only: INSERT only. No UPDATE. No DELETE.
type AgentSwitch struct {
	ID        string `db:"pk"` // set by caller via unixid before db.Create()
	IsEnabled bool   `db:"not_null"`
	ChangedBy string `db:"not_null"` // actor identity injected by the application layer
	Reason    string // optional free-text reason
}
