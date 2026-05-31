package providerpolicy

import "time"

const (
	DefaultFailureThreshold = 3
	DefaultCooldown         = 30 * time.Second
	DefaultTimeout          = 5 * time.Second
)

type Config struct {
	FailureThreshold int
	Cooldown         time.Duration
	Timeout          time.Duration
	MaxPayloadBytes  int
}

func Normalize(cfg Config) Config {
	cfg.FailureThreshold = FailureThreshold(cfg.FailureThreshold)
	cfg.Cooldown = Cooldown(cfg.Cooldown)
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	return cfg
}

func FailureThreshold(threshold int) int {
	if threshold <= 0 {
		return DefaultFailureThreshold
	}
	return threshold
}

func Cooldown(cooldown time.Duration) time.Duration {
	if cooldown <= 0 {
		return DefaultCooldown
	}
	return cooldown
}
