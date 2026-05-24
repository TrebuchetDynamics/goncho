package pluginruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	workspacepkg "github.com/TrebuchetDynamics/goncho/workspace"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	LocalHonchoAPIKeySentinel = "local"

	GonchoConfigBaseURLInvalid = "goncho_config_base_url_invalid"
	GonchoConfigLocalBaseURL   = "goncho_config_local_base_url"
)

type ConfigEvidence struct {
	Code    string `json:"code"`
	Source  string `json:"source,omitempty"`
	Message string `json:"message,omitempty"`
}

type PluginConfigInput struct {
	Host string
	Raw  map[string]any
	Env  map[string]string
}

type PluginConfig struct {
	Host              string
	WorkspaceID       string
	APIKey            string
	APIKeySource      string
	Environment       string
	BaseURL           string
	PeerName          string
	AIPeer            string
	PinPeerName       bool
	Enabled           bool
	SaveMessages      bool
	WriteFrequency    PluginWriteFrequency
	SessionStrategy   string
	SessionPeerPrefix bool
	Sessions          map[string]string
	Raw               map[string]any
	Evidence          []ConfigEvidence
}

func (c PluginConfig) HasEvidence(code string) bool {
	for _, item := range c.Evidence {
		if item.Code == code {
			return true
		}
	}
	return false
}

func ResolvePluginConfig(input PluginConfigInput) PluginConfig {
	host := strings.TrimSpace(input.Host)
	if host == "" {
		host = "hermes"
	}
	raw := input.Raw
	if raw == nil {
		raw = map[string]any{}
	}
	env := input.Env
	if env == nil {
		env = map[string]string{}
	}
	hostCfg := pluginHostBlock(raw, host)
	out := PluginConfig{
		Host:              host,
		WorkspaceID:       firstConfigString(hostCfg, raw, "workspace", host),
		Environment:       firstConfigString(hostCfg, raw, "environment", "production"),
		PeerName:          firstConfigString(hostCfg, raw, "peerName", ""),
		AIPeer:            firstConfigString(hostCfg, raw, "aiPeer", host),
		PinPeerName:       firstConfigBool(hostCfg, raw, "pinPeerName", false),
		SaveMessages:      firstConfigBool(hostCfg, raw, "saveMessages", true),
		SessionStrategy:   firstConfigString(hostCfg, raw, "sessionStrategy", "per-directory"),
		SessionPeerPrefix: firstConfigBool(hostCfg, raw, "sessionPeerPrefix", false),
		Sessions:          pluginSessionsMap(hostCfg, raw),
		Raw:               raw,
	}
	out.APIKey, out.APIKeySource = firstConfigStringWithSource(hostCfg, raw, env, "apiKey", "HONCHO_API_KEY")
	out.BaseURL, _ = firstConfigStringWithSource2(hostCfg, raw, env, []string{"baseUrl", "base_url"}, "HONCHO_BASE_URL")
	out.WriteFrequency = ParsePluginWriteFrequency(firstConfigAny(hostCfg, raw, "writeFrequency", "async"))

	baseURLInvalid := false
	if out.APIKey == "" && strings.TrimSpace(out.BaseURL) != "" {
		if plausibleHonchoBaseURL(out.BaseURL) {
			out.APIKey = LocalHonchoAPIKeySentinel
			out.APIKeySource = "base_url"
			out.Evidence = append(out.Evidence, ConfigEvidence{Code: GonchoConfigLocalBaseURL, Source: "base_url"})
		} else {
			baseURLInvalid = true
			out.Evidence = append(out.Evidence, ConfigEvidence{Code: GonchoConfigBaseURLInvalid, Source: "base_url"})
		}
	}
	out.Enabled = firstConfigBoolPointer(hostCfg, raw, "enabled", out.APIKey != "" || out.BaseURL != "")
	if baseURLInvalid && out.APIKey == "" {
		out.Enabled = false
	}
	return out
}

func pluginHostBlock(raw map[string]any, host string) map[string]any {
	hosts, ok := raw["hosts"].(map[string]any)
	if !ok {
		return nil
	}
	block, ok := hosts[host].(map[string]any)
	if !ok {
		return nil
	}
	return block
}

func firstConfigAny(hostCfg, raw map[string]any, key string, fallback any) any {
	if hostCfg != nil {
		if value, ok := hostCfg[key]; ok {
			return value
		}
	}
	if value, ok := raw[key]; ok {
		return value
	}
	return fallback
}

func firstConfigString(hostCfg, raw map[string]any, key, fallback string) string {
	value := firstConfigAny(hostCfg, raw, key, fallback)
	if s, ok := configString(value); ok {
		return s
	}
	return fallback
}

func firstConfigStringWithSource(hostCfg, raw map[string]any, env map[string]string, key, envKey string) (string, string) {
	if hostCfg != nil {
		if value, ok := configString(hostCfg[key]); ok && value != "" {
			return value, "host"
		}
	}
	if value, ok := configString(raw[key]); ok && value != "" {
		return value, "root"
	}
	if value := strings.TrimSpace(env[envKey]); value != "" {
		return value, "env"
	}
	return "", ""
}

func firstConfigStringWithSource2(hostCfg, raw map[string]any, env map[string]string, keys []string, envKey string) (string, string) {
	for _, key := range keys {
		if hostCfg != nil {
			if value, ok := configString(hostCfg[key]); ok && value != "" {
				return value, "host"
			}
		}
	}
	for _, key := range keys {
		if value, ok := configString(raw[key]); ok && value != "" {
			return value, "root"
		}
	}
	if value := strings.TrimSpace(env[envKey]); value != "" {
		return value, "env"
	}
	return "", ""
}

func firstConfigBool(hostCfg, raw map[string]any, key string, fallback bool) bool {
	return firstConfigBoolPointer(hostCfg, raw, key, fallback)
}

func firstConfigBoolPointer(hostCfg, raw map[string]any, key string, fallback bool) bool {
	if hostCfg != nil {
		if value, ok := configBool(hostCfg[key]); ok {
			return value
		}
	}
	if value, ok := configBool(raw[key]); ok {
		return value
	}
	return fallback
}

func configString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case fmt.Stringer:
		return strings.TrimSpace(v.String()), true
	default:
		return "", false
	}
}

func configBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	default:
		return false, false
	}
	return false, false
}

func pluginSessionsMap(hostCfg, raw map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range rawStringMap(raw["sessions"]) {
		out[k] = v
	}
	if hostCfg != nil {
		for k, v := range rawStringMap(hostCfg["sessions"]) {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func rawStringMap(value any) map[string]string {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for k, v := range raw {
		if s, ok := configString(v); ok && s != "" {
			out[k] = s
		}
	}
	return out
}

func plausibleHonchoBaseURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	switch lower {
	case "true", "false", "null", "localhost":
		return false
	}
	if _, err := strconv.Atoi(value); err == nil {
		return false
	}
	if strings.HasPrefix(lower, "file:") || strings.HasPrefix(lower, "ftp:") || strings.HasPrefix(lower, "ws:") {
		return false
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(value)
		return err == nil && parsed.Host != ""
	}
	return strings.Contains(value, ".") || strings.Contains(value, ":")
}

type SessionNameInput struct {
	CWD               string
	Title             string
	SessionID         string
	GatewaySessionKey string
}

func ResolvePluginSessionName(cfg PluginConfig, input SessionNameInput) string {
	cwd := strings.TrimSpace(input.CWD)
	if cwd == "" {
		cwd = "."
	}
	if cfg.Sessions != nil {
		if manual := strings.TrimSpace(cfg.Sessions[cwd]); manual != "" {
			return manual
		}
	}
	if title := sanitizePluginID(input.Title); title != "" {
		return withPluginPeerPrefix(cfg, title)
	}
	if key := sanitizePluginID(input.GatewaySessionKey); key != "" {
		return enforcePluginSessionIDLimit(key, input.GatewaySessionKey)
	}
	strategy := strings.TrimSpace(cfg.SessionStrategy)
	if strategy == "" {
		strategy = "per-directory"
	}
	if strategy == "per-session" && strings.TrimSpace(input.SessionID) != "" {
		return withPluginPeerPrefix(cfg, strings.TrimSpace(input.SessionID))
	}
	if strategy == "global" {
		if workspace := sanitizePluginID(cfg.WorkspaceID); workspace != "" {
			return workspace
		}
		return workspacepkg.DefaultWorkspaceID
	}
	return withPluginPeerPrefix(cfg, sanitizePluginID(filepath.Base(cwd)))
}

func withPluginPeerPrefix(cfg PluginConfig, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if cfg.SessionPeerPrefix && strings.TrimSpace(cfg.PeerName) != "" {
		return sanitizePluginID(cfg.PeerName) + "-" + value
	}
	return value
}

func enforcePluginSessionIDLimit(sanitized, original string) string {
	const maxLen = 120
	const hashLen = 12
	if len(sanitized) <= maxLen {
		return sanitized
	}
	sum := sha256.Sum256([]byte(original))
	digest := hex.EncodeToString(sum[:])[:hashLen]
	prefixLen := maxLen - hashLen - 1
	return strings.TrimRight(sanitized[:prefixLen], "-") + "-" + digest
}

type PluginPeerInput struct {
	Config              PluginConfig
	RuntimeUserPeerName string
	SessionKey          string
}

type PluginPeerResolution struct {
	UserPeerID      string
	AssistantPeerID string
}

func ResolvePluginPeerNames(input PluginPeerInput) PluginPeerResolution {
	cfg := input.Config
	pinPeerName := cfg.PinPeerName && strings.TrimSpace(cfg.PeerName) != ""
	var user string
	switch {
	case strings.TrimSpace(input.RuntimeUserPeerName) != "" && !pinPeerName:
		user = input.RuntimeUserPeerName
	case strings.TrimSpace(cfg.PeerName) != "":
		user = cfg.PeerName
	default:
		parts := strings.SplitN(input.SessionKey, ":", 2)
		channel := "default"
		chat := input.SessionKey
		if len(parts) == 2 {
			channel = parts[0]
			chat = parts[1]
		}
		user = "user-" + channel + "-" + chat
	}
	assistant := strings.TrimSpace(cfg.AIPeer)
	if assistant == "" {
		assistant = "hermes-assistant"
	}
	return PluginPeerResolution{
		UserPeerID:      sanitizePluginID(user),
		AssistantPeerID: sanitizePluginID(assistant),
	}
}

var pluginIDUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizePluginID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = pluginIDUnsafe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}
