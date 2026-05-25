package memorymirror

// CompatCatalog documents the agentmemory-style alias layer Goncho is willing
// to expose. It is separate from NewToolRegistry so broad upstream tools can be
// classified without automatically enabling every mutating operation.
type CompatCatalog struct {
	Tools []CompatTool `json:"tools"`
}

type CompatTool struct {
	Name           string     `json:"name"`
	RegisteredName string     `json:"registered_name,omitempty"`
	Status         PortStatus `json:"status"`
	GonchoSeam     string     `json:"goncho_seam"`
	Mutating       bool       `json:"mutating"`
	DefaultEnabled bool       `json:"default_enabled"`
	Residual       string     `json:"residual,omitempty"`
}

func CompatibilityCatalog() CompatCatalog {
	return CompatCatalog{Tools: []CompatTool{
		compatTool("memory_save", "memory_save", PortDelivered, "service.Service.Conclude", true, true, ""),
		compatTool("memory_smart_search", "memory_smart_search", PortDelivered, "service.Service.Search", false, true, ""),
		compatTool("memory_recall", "memory_recall", PortDelivered, "service.Service.Recall", false, true, ""),
		compatTool("memory_profile", "memory_profile", PortDelivered, "service.Service.Profile", false, true, ""),
		compatTool("memory_timeline", "memory_timeline", PortDelivered, "service.ViewerSessionTimeline", false, true, "read-only JSON timeline over messages, observations, and summaries"),
		compatTool("memory_audit", "memory_audit", PortDelivered, "service.AuditTrail", false, true, "read-only audit trail over observation-backed events"),
		compatTool("memory_slot_list", "", PortPartial, "service.ListMemorySlots", false, false, "slot aliases are implemented in service but not enabled in default compatibility registry yet"),
		compatTool("memory_slot_get", "", PortPartial, "service.GetMemorySlot", false, false, "slot aliases are implemented in service but not enabled in default compatibility registry yet"),
		compatTool("memory_snapshot_create", "", PortAdapter, "service.ExportSnapshotManifest", false, false, "manifest export is deterministic; git operations stay adapter-owned"),
		compatTool("memory_graph_query", "", PortPartial, "service.Service.Recall graph provenance", false, false, "dedicated graph query projection remains deferred behind recall trace"),
		compatTool("memory_verify", "", PortPartial, "service.Recall provenance + live-check warnings", false, false, "dedicated verification projection remains deferred behind recall trace"),
		compatTool("memory_diagnose", "", PortPartial, "service diagnostics + queue status", false, false, "diagnostic projection exists but is not enabled as a default alias yet"),
	}}
}

func (c CompatCatalog) CompatTool(name string) (CompatTool, bool) {
	needle := normalize(name)
	for _, tool := range c.Tools {
		if normalize(tool.Name) == needle || normalize(tool.RegisteredName) == needle {
			return tool, true
		}
	}
	return CompatTool{}, false
}

func compatTool(name, registeredName string, status PortStatus, seam string, mutating, defaultEnabled bool, residual string) CompatTool {
	return CompatTool{Name: name, RegisteredName: registeredName, Status: status, GonchoSeam: seam, Mutating: mutating, DefaultEnabled: defaultEnabled, Residual: residual}
}
