package config

import "context"

type contextKey string

const configSnapshotKey contextKey = "config_snapshot"

func WithConfigSnapshot(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configSnapshotKey, cfg)
}

func GetConfigSnapshot(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configSnapshotKey).(*Config); ok {
		return cfg
	}
	return nil
}
