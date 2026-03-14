# Migrate to tinywasm/orm v2 API (fmt.Field)

## Context

The ORM code generator (`ormc`) now produces `Schema() []fmt.Field` (from `tinywasm/fmt`) with individual bool constraint fields instead of the old `[]orm.Field` with bitmask constraints. The `Values()` method is removed; consumers use `fmt.ReadValues(schema, ptrs)` instead.

### Key API Changes

| Old (current) | New (target) |
|---|---|
| `[]orm.Field{Name, Type: orm.TypeText, Constraints: orm.ConstraintPK}` | `[]fmt.Field{Name, Type: fmt.FieldText, PK: true}` |
| `orm.TypeText`, `orm.TypeBool` | `fmt.FieldText`, `fmt.FieldBool` |
| `orm.ConstraintPK`, `orm.ConstraintNotNull` (bitmask) | `PK: true`, `NotNull: true` (bool fields) |
| `m.Values() []any` | `fmt.ReadValues(m.Schema(), m.Pointers())` |
| `var AgentSwitchMeta = struct{...}` | `var AgentSwitch_ = struct{...}` (standardized `_` suffix) |

### Target fmt.Field Struct (`tinywasm/fmt`)

```go
type Field struct {
    Name    string
    Type    FieldType // FieldText, FieldInt, FieldFloat, FieldBool, FieldBlob, FieldStruct
    PK      bool
    Unique  bool
    NotNull bool
    AutoInc bool
    Input   string
    JSON    string
}
```

### Generated Code per Struct (`ormc`)

- `TableName() string`, `FormName() string`
- `Schema() []fmt.Field`, `Pointers() []any`
- `T_` metadata struct with typed column constants
- `ReadOneT(qb *orm.QB, model *T)`, `ReadAllT(qb *orm.QB)`

---

## Stage 1 — Regenerate ORM Code

**File**: `model_orm.go` (auto-generated)

1. Update `ormc`: `go install github.com/tinywasm/orm/cmd/ormc@latest`
2. Run `ormc` from project root
3. Verify generated file uses `fmt.Field` with bool constraints
4. Verify meta struct uses `_` suffix: `AgentSwitch_` (not `AgentSwitchMeta`)
5. Verify `Values()` is no longer generated

---

## Stage 2 — Update Handwritten Code

**File**: `mcp.go`

1. Replace `AgentSwitchMeta.ID` → `AgentSwitch_.ID` (and all other meta references)
2. Replace `AgentSwitchMeta.TableName` → `AgentSwitch_.TableName` if used
3. Search for `.Values()` calls → replace with `fmt.ReadValues(m.Schema(), m.Pointers())`
4. Add `"github.com/tinywasm/fmt"` import if needed for `ReadValues`

> **Note**: `db.Query()`, `.Where()`, `.OrderBy()`, `.Desc()`, `ReadAllAgentSwitch()` — all unchanged.

---

## Stage 3 — Update go.mod

1. Run `go mod tidy`
2. Ensure `tinywasm/fmt` and `tinywasm/orm` are at latest versions

---

## Verification

```bash
gotest
```

## Linked Documents

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [SKILL.md](SKILL.md)
