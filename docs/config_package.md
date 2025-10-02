# Config Package Documentation

## Overview

The `config` package handles loading, validating, and managing all configuration for TrafficCTRL. It supports both YAML files and environment variable overrides.

---

## Files

### **types.go**

Defines all configuration structures and custom types.

**Key Types:**

- `Config` - Root struct that holds all config (Proxy, Limiter, Redis, Logger)
- `ProxyConfig` - Target URL, ports, server name, dry run mode
- `RedisConfig` - Redis connection settings
- `LoggerConfig` - Log level, environment, output path
- `RateLimiterConfig` - Global, PerTenant, and PerEndpoint rate limiting rules
- `AlgorithmConfig` - Algorithm type and its parameters (capacity, rates, windows, etc.)
- `EndpointRule` - Path-specific rate limiting rules with wildcard support
- `TenantStrategy` - How to identify users (IP, header, cookie, query param)

**Custom Types:**

- `Duration` - Wraps `time.Duration` with custom YAML unmarshaling to parse strings like `"60s"`, `"5m"`
- `AlgorithmType` - Enum for the 4 algorithms: `token_bucket`, `leaky_bucket`, `fixed_window`, `sliding_window`
- `TenantStrategyType` - Enum: `ip`, `header`, `cookie`, `query_parameter`

**Note:** Most fields in `AlgorithmConfig` are pointers (`*int`, `*Duration`) to distinguish between "not set" (nil) and "set to zero".

---

### **loader.go**

Generic YAML file loader using Go generics.

**Main Function:**

```go
func loadFromFile[T any](path string) (*T, error)
```

- Opens a YAML file
- Decodes it into type `T`
- Returns pointer to parsed config or error

Simple, reusable for any YAML config file.

---

### **init.go**

Orchestrates loading all configs and applying environment variable overrides.

**Main Function:**

```go
func LoadConfigs() (*Config, error)
```

Loads all 4 config files in order:

1. Logger (so we can log errors from other configs)
2. Redis
3. Proxy
4. Limiter

Returns aggregated `Config` struct.

**Individual Loaders:**

- `loadLoggerConfig()` - Loads `logger.yaml`, overrides: `LOG_LEVEL`, `LOG_ENVIRONMENT`, `LOG_OUTPUT_PATH`
- `loadRedisConfig()` - Loads `redis.yaml`, overrides: `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_POOL_SIZE`, `REDIS_USE_TLS`, `REDIS_TLS_SKIP_VERIFY`
- `loadProxyConfig()` - Loads `proxy.yaml`, overrides: `TARGET_URL`, `PROXY_PORT`, `METRICS_PORT`, `DRY_RUN_MODE`
- `loadLimiterConfig()` - Loads `limiter.yaml`, **NO env overrides** (too complex)

**Helper Functions:**

- `getConfigPath(file)` - Builds config file path, respects `CONFIG_DIR` env var (default: `./config`)
- `parsePortEnv(envVar)` - Safely parses port from env var
- `parseIntEnv(envVar)` - Safely parses int from env var

**Note:** Each loader calls `validate()` before returning.

---

### **validator.go**

Validation logic for all config types. Called during initialization to catch errors early.

**Main Validators:**

**`ProxyConfig.validate()`**

- `target_url`: not empty, valid URL, has scheme (http/https only)
- Ports: 1024-65535 range, proxy_port ≠ metrics_port
- `server_name`: not empty

**`RedisConfig.validate()`**

- `address`: not empty
- `db`: >= 0
- `pool_size`: > 0

**`RateLimiterConfig.validate()`**

- Validates Global config if enabled
- Validates PerTenant config if enabled
- Validates each EndpointRule
- Warns on duplicate paths (doesn't fail, first match wins)

**`AlgorithmConfig.validate()`**

- Checks algorithm type is valid
- Dispatches to algorithm-specific validator:
  - `validateTokenBucket()` - requires: capacity, refill_rate, refill_period (all > 0)
  - `validateLeakyBucket()` - requires: capacity, leak_rate, leak_period (all > 0)
  - `validateFixedWindow()` - requires: window_size, limit (both > 0)
  - `validateSlidingWindow()` - requires: window_size, limit (both > 0)

**`TenantStrategy.validate()`**

- Type must be: `ip`, `header`, `cookie`, or `query_parameter`
- If type is NOT `ip`, `key` field is required

**`EndpointRule.validate()`**

- `path`: not empty
- If `bypass: true`, skip other checks
- HTTP methods: uppercase, must be valid (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
- Validates tenant_strategy if present
- Validates algorithm config

**`LoggerConfig.validate()`**

- `level`: one of `trace`, `debug`, `info`, `warn`, `error`, `fatal`
- `environment`: `development` or `production`

---

### **context.go**

Stores config in `context.Context` for thread-safe access across middleware/handlers.

**Functions:**

```go
func WithConfigSnapshot(ctx context.Context, cfg *Config) context.Context
```

Attaches config to context.

```go
func GetConfigFromContext(ctx context.Context) *Config
```

Retrieves config from context (returns nil if not found).

**Why:** Avoids global variables, thread-safe, idiomatic Go.

---

## Configuration Files

### **proxy.yaml**

```yaml
target_url: "http://localhost:5000" # Backend to proxy to
proxy_port: 8080 # Main proxy server port
metrics_port: 8090 # Prometheus metrics port
server_name: "trafficctrl:v0.1.0" # Server header
dry_run_mode: false # If true, log violations but don't block
```

### **redis.yaml**

```yaml
address: "localhost:6379" # Redis host:port
password: "" # Optional password
db: 0 # Database index (0-15)
pool_size: 40 # Max connection pool size
use_tls: false # Enable TLS
tls_skip_verify: false # Skip cert verification (insecure!)
```

### **logger.yaml**

```yaml
level: "debug" # trace|debug|info|warn|error|fatal
environment: "development" # development (human) | production (JSON)
output_path: "stdout" # stdout, stderr, or file path
```

### **limiter.yaml**

Three-layer rate limiting system with extensive comments. See the file for full examples.

**Structure:**

```yaml
global: # System-wide limit (triggers reputation system)
  enabled: true
  algorithm: token_bucket
  capacity: 20000
  refill_rate: 10000
  refill_period: "1m"

per_tenant: # Per-user limit across all endpoints
  enabled: true
  algorithm: sliding_window
  window_size: "1m"
  limit: 100

per_endpoint: # Per-user per-endpoint limits
  rules:
    - path: "/api/v1/auth/login"
      methods: ["POST"]
      tenant_strategy:
        type: ip
      algorithm: fixed_window
      window_size: "1m"
      limit: 10

    - path: "/api/v1/*" # Wildcard matching
      tenant_strategy:
        type: header
        key: "Authorization"
      algorithm: token_bucket
      capacity: 10
      refill_rate: 10
      refill_period: "1m"

    - path: "/health"
      bypass: true # Skip rate limiting completely

    - path: "*" # Catch-all
      algorithm: sliding_window
      window_size: "1m"
      limit: 20
```

**Key Features:**

- Wildcard path matching: `/api/*` matches all `/api/...` paths
- Rules evaluated in order, first match wins
- Each endpoint can have its own algorithm and tenant strategy
- `bypass: true` skips rate limiting entirely

---

## Usage Flow

1. **Application starts** → calls `LoadConfigs()`
2. **LoadConfigs()** → loads all 4 YAML files, applies env overrides, validates everything
3. **Validation fails** → app exits with error
4. **Validation succeeds** → returns `*Config`
5. **Config stored in context** → `WithConfigSnapshot(ctx, cfg)`
6. **Middleware/handlers** → retrieve with `GetConfigFromContext(ctx)`

---

## Notes

- **Environment variables override YAML** for most settings (except limiter config)
- **Validation happens at startup** - fail fast if config is invalid
- **Port range 1024-65535** - avoids privileged ports requiring root
- **Pointer fields in AlgorithmConfig** - distinguish "not set" from "zero value"
- **Limiter config** - too complex for env vars, must use YAML
- **Path matching** - first matching rule wins, use `*` for catch-all
