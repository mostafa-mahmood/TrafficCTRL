# TrafficCTRL Project Structure

```
TrafficCTRL/
├── cmd/
│   └── ctrl/
│       └── main.go                    # Application entry point
│
├── config/                            # Configuration package
│   ├── context.go                     # Config context propagation
│   ├── init.go                        # Config loader orchestrator
│   ├── loader.go                      # Generic YAML file loader
│   ├── types.go                       # Config type definitions
│   ├── validator.go                   # Config validation logic
│   ├── limiter.yaml                   # Rate limiting rules
│   ├── logger.yaml                    # Logging configuration
│   ├── proxy.yaml                     # Proxy server settings
│   └── redis.yaml                     # Redis connection settings
│
├── internal/                          # Private application code
│   ├── limiter/                       # Rate limiting algorithms
│   │   ├── client.go                  # Redis client wrapper
│   │   ├── limiter.go                 # Main limiter interface
│   │   ├── token_bucket.go            # Token bucket algorithm
│   │   ├── leaky_bucket.go            # Leaky bucket algorithm
│   │   ├── fixed_window.go            # Fixed window counter
│   │   ├── sliding_window.go          # Sliding window log
│   │   ├── reputation.go              # Reputation system
│   │   └── *_test.go                  # Unit tests
│   │
│   ├── logger/                        # Logging utilities
│   │   └── logger.go                  # Zap logger setup
│   │
│   ├── middleware/                    # HTTP middleware chain
│   │   ├── classifier.go              # Request classification
│   │   ├── keys.go                    # Redis key generation
│   │   ├── metadata.go                # Request metadata extraction
│   │   ├── tenant_limit.go            # Per-tenant rate limiting
│   │   ├── endpoint_limit.go          # Per-endpoint rate limiting
│   │   ├── global_limit.go            # Global rate limiting
│   │   ├── response.go                # Response helpers
│   │   ├── request_logger.go          # Request/response logging
│   │   ├── recover.go                 # Panic recovery
│   │   └── dry_run.go                 # Dry run mode handler
│   │
│   ├── proxy/                         # Reverse proxy
│   │   ├── proxy.go                   # HTTP reverse proxy logic
│   │   └── server.go                  # HTTP server setup
│   │
│   └── shared/                        # Shared utilities
│       ├── map.go                     # Thread-safe map
│       ├── tenant_parser.go           # Tenant ID extraction
│       └── sanitize_test.go           # Sanitization tests
│
├── metrics/                           # Prometheus metrics
├── test/                              # Integration/e2e tests
├── design/                            # Design assets (logos, diagrams)
├── docs/                              # Documentation
├── go.mod                             # Go module definition
├── go.sum                             # Dependency checksums
├── Dockerfile                         # Container build instructions
├── README.md                          # Project overview
└── LICENSE                            # MIT License
```
