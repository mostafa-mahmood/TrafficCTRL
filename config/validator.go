package config

import "fmt"

type Validator interface {
	Validate() error
}

func (l *LoggerConfigsType) Validate() error {
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal"}

	valid := false
	for _, level := range validLevels {
		if l.Level == level {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid logger config (level): %s, must be one of %v", l.Level, validLevels)
	}

	if l.Environment != "development" && l.Environment != "production" {
		return fmt.Errorf("invalid logger config (environment): %s, must be %s or %s", l.Environment, "development", "production")
	}

	return nil
}

func (p *ProxyConfigsType) Validate() error {
	if p.TargetUrl == "" {
		return fmt.Errorf("invalid proxy config (target_url): cannot be empty")
	}

	if p.ProxyPort == 0 {
		return fmt.Errorf("invalid proxy config (proxy_port): cannot be zero")
	}

	return nil
}
