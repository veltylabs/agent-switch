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

// ormc:formonly
type statusEmptyResult struct {
	IsEnabled bool
	ChangedAt int64
}

// ormc:formonly
type statusResult struct {
	IsEnabled bool
	ChangedBy string
	ChangedAt int64
	Reason    string
}

// ormc:formonly
// toggleArgs holds the incoming JSON parameters for the Toggle handler.
// IsEnabled uses plain bool; both true and false are valid toggle values.
// ChangedBy is required (db:"not_null").
type toggleArgs struct {
	IsEnabled bool
	ChangedBy string `db:"not_null"`
	Reason    string
}

// ormc:formonly
type toggleResult struct {
	OK        bool
	IsEnabled bool
}
