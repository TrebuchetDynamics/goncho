package hostintegration

import (
	"path/filepath"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// HookBundlePlan describes a generated hook artifact without writing it. Host
// connectors can render these plans into host-specific scripts only after the
// operator reviews the output and host smoke coverage exists.
type HookBundlePlan struct {
	Host           string         `json:"host"`
	Event          string         `json:"event"`
	Command        []string       `json:"command"`
	PayloadSchema  map[string]any `json:"payload_schema"`
	RedactionClass string         `json:"redaction_class"`
	OutputPath     string         `json:"output_path"`
	InstallStatus  string         `json:"install_status"`
}

// BuildHookBundlePlan returns non-mutating hook bundle plans for every
// host-neutral event accepted by service.CaptureHostHook.
func BuildHookBundlePlan(host, outputDir string) []HookBundlePlan {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "generic"
	}
	outputDir = strings.TrimSpace(outputDir)
	if outputDir == "" {
		outputDir = filepath.Join(".", "goncho-hooks", host)
	}
	schemas := goncho.HostHookEventSchemas()
	plans := make([]HookBundlePlan, 0, len(schemas))
	for _, schema := range schemas {
		event := string(schema.Event)
		plans = append(plans, HookBundlePlan{
			Host:           host,
			Event:          event,
			Command:        []string{"goncho-hook", "capture", "--host", host, "--event", event},
			PayloadSchema:  schema.JSONSchema,
			RedactionClass: "strict_host_hook_redaction",
			OutputPath:     filepath.Join(outputDir, event+".json"),
			InstallStatus:  "plan_only",
		})
	}
	return plans
}
