package toolmeta

import (
	"context"
	"encoding/json"
	"time"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Timeout() time.Duration
	Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

type Spec interface {
	Spec() OperationSpec
}

type ToolDescriptor struct {
	Name, Description string
	Schema            json.RawMessage
}

type OperationSpec struct {
	ToolDescriptor
	Mutating, Idempotent, PromptSafe bool
	TrustClass                       []string
	AuditKind                        string
}

func DefaultSpec(name, desc string, schema json.RawMessage) OperationSpec {
	return OperationSpec{ToolDescriptor: ToolDescriptor{Name: name, Description: desc, Schema: schema}, Mutating: true, Idempotent: false, PromptSafe: true, TrustClass: []string{"operator", "child-agent", "system"}, AuditKind: "tool"}
}

var memoryToolDescriptors = []ToolDescriptor{
	{Name: "store_memory", Description: "Persist information to agent memory.", Schema: json.RawMessage(`{"type":"object","properties":{"content":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}},"importance":{"type":"number"},"metadata":{"type":"object","additionalProperties":{"type":"string"}}},"required":["content"]}`)},
	{Name: "retrieve_memory", Description: "Retrieve memories relevant to the query.", Schema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},"required":["query"]}`)},
	{Name: "update_memory", Description: "Update an existing memory entry.", Schema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"content":{"type":"string"},"importance":{"type":"number"}},"required":["id"]}`)},
	{Name: "summarize_memories", Description: "Summarize related memories.", Schema: json.RawMessage(`{"type":"object","properties":{"filter":{"type":"string"},"max_items":{"type":"integer"}},"required":["filter"]}`)},
	{Name: "forget_memory", Description: "Remove a memory entry (soft delete).", Schema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`)},
	{Name: "goncho_review", Description: "Inspect and resolve Goncho memory review items.", Schema: json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["list","resolve"]},"peer_id":{"type":"string"},"status":{"type":"string","enum":["open","resolved"]},"id":{"type":"string"},"resolution":{"type":"string","enum":["accepted","rejected","superseded","verified"]},"resolved_by":{"type":"string"},"resolution_reason":{"type":"string"}},"required":["action"]}`)},
}

func MemoryToolOperationSpec(name string) (OperationSpec, bool) {
	for _, d := range memoryToolDescriptors {
		if d.Name == name {
			s := OperationSpec{ToolDescriptor: d, Mutating: true, Idempotent: false, PromptSafe: true, TrustClass: []string{"operator", "child-agent", "system"}, AuditKind: "memory"}
			switch d.Name {
			case "retrieve_memory", "summarize_memories":
				s.Mutating = false
				s.Idempotent = true
			case "forget_memory":
				s.Idempotent = true
			case "goncho_review":
				s.AuditKind = "review"
				s.TrustClass = []string{"operator", "system"}
			}
			return s, true
		}
	}
	return OperationSpec{}, false
}
