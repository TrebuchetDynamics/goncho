package goncho

import pluginruntime "github.com/TrebuchetDynamics/goncho/internal/pluginruntime"

const (
	LocalHonchoAPIKeySentinel = pluginruntime.LocalHonchoAPIKeySentinel

	GonchoConfigBaseURLInvalid = pluginruntime.GonchoConfigBaseURLInvalid
	GonchoConfigLocalBaseURL   = pluginruntime.GonchoConfigLocalBaseURL
)

type ConfigEvidence = pluginruntime.ConfigEvidence

type PluginConfigInput = pluginruntime.PluginConfigInput

type PluginConfig = pluginruntime.PluginConfig

type SessionNameInput = pluginruntime.SessionNameInput

type PluginPeerInput = pluginruntime.PluginPeerInput

type PluginPeerResolution = pluginruntime.PluginPeerResolution

func ResolvePluginConfig(input PluginConfigInput) PluginConfig {
	return pluginruntime.ResolvePluginConfig(input)
}

func ResolvePluginSessionName(cfg PluginConfig, input SessionNameInput) string {
	return pluginruntime.ResolvePluginSessionName(cfg, input)
}

func ResolvePluginPeerNames(input PluginPeerInput) PluginPeerResolution {
	return pluginruntime.ResolvePluginPeerNames(input)
}
