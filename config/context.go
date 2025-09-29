package config

import "context"

type contextKey struct{}

var configSnapshotKey = contextKey{}

func WithConfigSnapshot(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configSnapshotKey, cfg)
}

func GetConfigFromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configSnapshotKey).(*Config); ok {
		return cfg
	}
	return nil
}
