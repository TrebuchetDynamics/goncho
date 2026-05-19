package goncho

import "encoding/json"

// OperationSpec describes a memory tool operation for MCP registration.
type OperationSpec struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema"`
	Schema       json.RawMessage `json:"-"` // alias for InputSchema in tests
	ToolDescriptor
	AuditKind  string   `json:"audit_kind,omitempty"`
	PromptSafe bool     `json:"prompt_safe,omitempty"`
	TrustClass []string `json:"trust_class,omitempty"`
	Mutating   bool     `json:"mutating,omitempty"`
	Idempotent bool     `json:"idempotent,omitempty"`
}

// ToolDescriptor holds MCP tool registration metadata.
type ToolDescriptor struct {
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// MemoryToolOperationSpec returns the canonical operation spec for a named
// memory tool, or false if no canonical spec exists.
func MemoryToolOperationSpec(name string) (OperationSpec, bool) {
	specs := map[string]OperationSpec{
		"store_memory": {
			Name:         "store_memory",
			Description:  "Persist information to agent memory.",
			ToolDescriptor: ToolDescriptor{Category: "memory"},
			AuditKind:    "memory",
			PromptSafe:   true,
			TrustClass:   []string{"operator", "system"},
			Mutating:     true,
			Idempotent:   false,
		},
		"retrieve_memory": {
			Name:         "retrieve_memory",
			Description:  "Search and retrieve memories.",
			ToolDescriptor: ToolDescriptor{Category: "memory", ReadOnly: true},
			AuditKind:    "memory",
			PromptSafe:   true,
			TrustClass:   []string{"operator", "system"},
			Mutating:     false,
			Idempotent:   true,
		},
		"update_memory": {
			Name:         "update_memory",
			Description:  "Update an existing memory entry.",
			ToolDescriptor: ToolDescriptor{Category: "memory"},
			AuditKind:    "memory",
			PromptSafe:   true,
			TrustClass:   []string{"operator", "system"},
			Mutating:     true,
			Idempotent:   false,
		},
		"summarize_memories": {
			Name:         "summarize_memories",
			Description:  "Generate a summary of multiple memories.",
			ToolDescriptor: ToolDescriptor{Category: "memory"},
			AuditKind:    "memory",
			PromptSafe:   true,
			TrustClass:   []string{"operator", "system"},
			Mutating:     false,
			Idempotent:   true,
		},
		"forget_memory": {
			Name:         "forget_memory",
			Description:  "Soft-delete a memory entry.",
			ToolDescriptor: ToolDescriptor{Category: "memory"},
			AuditKind:    "memory",
			PromptSafe:   true,
			TrustClass:   []string{"operator", "system"},
			Mutating:     true,
			Idempotent:   true,
		},
	}
	spec, ok := specs[name]
	return spec, ok
}

// DefaultSpec returns a default operation spec when no canonical spec exists.
func DefaultSpec(name, description string, schema json.RawMessage) OperationSpec {
	return OperationSpec{
		Name:         name,
		Description:  description,
		InputSchema:  schema,
		ToolDescriptor: ToolDescriptor{Category: "memory"},
		AuditKind:    "memory",
		PromptSafe:   true,
		TrustClass:   []string{"operator", "system"},
	}
}
