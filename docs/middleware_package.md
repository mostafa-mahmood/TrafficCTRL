# Middleware Package Documentation

## Overview

The `middleware` package is the **request processing pipeline** for TrafficCTRL. It organizes the admission control logic into a configurable chain of standard `http.Handler` functions.

It is responsible for:

1.  **Metadata Injection**: Setting request-specific identifiers and client IP addresses.
2.  **Request Classification**: Matching the incoming request to a specific rate-limiting rule.
3.  **Execution**: Running the three layers of admission control (**Global**, **Per-Tenant**, **Per-Endpoint**) in the correct sequence.
4.  **Error Handling & Observability**: Providing panic recovery, request-scoped logging, metric tracking, and standardized rejection responses.

---

## Files

### **keys.go**

Defines the context keys used to pass request-specific data between middlewares in the chain.

**Key Type:**

```go
type ctxKey string
```

**Constants (Context Keys):**

| Constant           | Description                                                   |
| :----------------- | :------------------------------------------------------------ |
| `RequestIDKey`     | The unique `X-Request-ID` generated or received.              |
| `ClientIPKey`      | The extracted client IP (`X-Real-IP`).                        |
| `EndpointRuleKey`  | The matched `config.EndpointRule` for the request.            |
| `TenantKeyKey`     | The extracted unique tenant identifier (e.g., user ID, IP).   |
| `RequestLoggerKey` | The request-scoped logger instance.                           |
| `RedisContextKey`  | A context with a short timeout for Redis operations.          |
| `BypassKey`        | A boolean flag indicating if rate limiting should be skipped. |

**Key Functions:**

```go
func IsBypassEnabled(ctx context.Context) bool
```

Checks the context for the `BypassKey`. If set to `true`, the subsequent limiting middlewares will skip execution.

---

### **metadata.go**

The first middleware in the chain. It ensures every request has essential identifying information for logging and rate limiting.

**Key Function:**

```go
func MetadataMiddleware(next http.Handler) http.Handler
```

**Function Logic:**

1.  **Request ID**: Checks the `X-Request-ID` header. If missing, a new `uuid` is generated and set on the request header and context.
2.  **Client IP**: Extracts the client IP from standard headers (like `X-Forwarded-For` or `X-Real-IP`). If `X-Real-IP` is not set, it is set with the extracted IP.
3.  **Context**: Adds both the Request ID and Client IP to the request context.

---

### **request_logger.go**

Defines a specialized logger (`requestLogger`) that automatically enriches logs with request metadata.

**Key Type:**

```go
type requestLogger struct {
    *logger.Logger
    baseFields []zap.Field
}
```

**Logic:**

The `requestLogger` wraps the base Zap logger and embeds **request_id**, **client_ip**, **path**, **method**, and **host** fields into every log message (Debug, Info, Warn, Error). This ensures complete observability for every single request.

---

### **classifier.go**

The core logic for classifying the request and setting up the environment for the limiting chain.

**Key Function:**

```go
func ClassifierMiddleware(next http.Handler, lgr *logger.Logger) http.Handler
```

**Function Logic:**

1.  **Instantiate Logger**: Creates the request-scoped `requestLogger` and attaches it to the context.
2.  **Match Rule**: Maps the incoming request (path/method) to the correct `config.EndpointRule` defined in `limiter.yaml`.
3.  **Bypass Check**: If no rule is matched or the matched rule has the `Bypass` flag set, a `BypassKey` is set on the context, and the request is allowed to proceed down the chain (which will skip all limit checks).
4.  **Extract Tenant Key**: If not bypassed, the unique **tenant key** (e.g., user ID, IP) is extracted based on the `TenantStrategy` defined in the matched rule, and attached to the context.
5.  **Redis Context**: Attaches a new context for Redis operations (`RedisContextKey`) to ensure predictable timeouts.

---

### **global_limit.go**

Enforces the **Global Limit** (System High Load Detection) layer.

**Key Function:**

```go
func GlobalLimitMiddleware(next http.Handler, lgr *logger.Logger,
	rateLimiter *limiter.RateLimiter) http.Handler
```

**Function Logic:**

1.  **Check Bypass**: If bypass is enabled, proceed.
2.  **Check Configuration**: If global limiting is disabled, proceed.
3.  **Check Limit**: Calls `rateLimiter.CheckGlobalLimit()`.
4.  **High Load Handling (Reputation Check)**:
    - If the global limit is **exceeded**, the request is not immediately denied.
    - The middleware fetches the **tenant's Reputation Score**.
    - If the `reputation.Score` is less than or equal to the minimum threshold (currently 0.3), the request is **rejected** with a specific message (`rejectBadReputationTenant`).
    - If the reputation check passes, the request is allowed to proceed, even though the system is under high load (fail-open for good users).
5.  **Metrics**: Tracks `GlobalLimitErrors` and observes the `ReputationDistribution`.

---

### **tenant_limit.go**

Enforces the **Per-Tenant Limit** (Fair Share) layer.

**Key Function:**

```go
func TenantLimitMiddleware(next http.Handler, rateLimiter *limiter.RateLimiter) http.Handler
```

**Function Logic:**

1.  **Check Bypass/Config**: Skips if bypass is active or if per-tenant limiting is disabled.
2.  **Check Limit**: Calls `rateLimiter.CheckTenantLimit()` using the extracted tenant key.
3.  **Rejection**:
    - If **not allowed**, the request is immediately rejected using `rejectRequest()`.
    - **Reputation Update**: The tenant's reputation is penalized (`UpdateReputation(..., true)`).
4.  **Metrics**: Tracks `TenantLimitErrors`.

---

### **endpoint_limit.go**

Enforces the **Per-Endpoint Limit** (Granular protection) layer.

**Key Function:**

```go
func EndpointLimitMiddleware(next http.Handler, rateLimiter *limiter.RateLimiter) http.Handler
```

**Function Logic:**

1.  **Check Bypass**: Skips if bypass is active.
2.  **Check Limit**: Calls `rateLimiter.CheckEndpointLimit()` using the extracted tenant key and matched endpoint rule.
3.  **Rejection**:
    - If **not allowed**, the request is immediately rejected.
    - **Reputation Update**: The tenant's reputation is penalized (`UpdateReputation(..., true)`).
4.  **Success/Final Actions**:
    - If **allowed**, the request is counted as an `AllowedRequests` metric.
    - **Reputation Update**: The tenant's reputation is rewarded (`UpdateReputation(..., false)`).

---

### **dry_run.go**

Implements the optional Dry Run mode for testing policies.

**Key Function:**

```go
func DryRunMiddleware(next http.Handler, rateLimiter *limiter.RateLimiter) http.Handler
```

**Function Logic:**

1.  **Check Config**: Only runs if `Proxy.DryRunMode` is enabled.
2.  **Check All Limits**: Unlike the enforcement middlewares, this middleware runs all three limit checks (`Global`, `Per-Tenant`, `Per-Endpoint`) **without enforcing the denial logic**.
3.  **Logging**: If any limit check is exceeded, a `WARN` message is logged stating that the limit **would have been exceeded (dry run)**, along with the calculated `retry_after` time.
4.  **Pass-Through**: In all cases, the request is forwarded to the `next` handler (the backend).

---

### **response.go**

Handles all standard rejection responses (HTTP 429).

**Key Functions:**

```go
func rejectRequest(res http.ResponseWriter, reqLogger *requestLogger, result *limiter.LimitResult,
	limitLevel config.LimitLevelType)
```

**Purpose**: The standard rejection response for **Per-Tenant** and **Per-Endpoint** limit violations.

**Response Headers/Body:**

- **Status Code**: `429 Too Many Requests`.
- **Header**: Sets `Content-Type: application/json`.
- **Header**: Sets `X-RateLimit-Remaining: 0`.
- **Header**: Sets `Retry-After` header using the `result.RetryAfter` value in seconds.
- **Body**: JSON payload includes `error: "rate limit exceeded"`, `limit_level`, `remaining`, and `retry_after`.
- **Metrics**: Increments `metrics.DeniedRequests` with the corresponding `limit_level` label.

<!-- end list -->

```go
func rejectBadReputationTenant(res http.ResponseWriter, reqLogger *requestLogger,
	reputation *limiter.Reputation, result *limiter.LimitResult)
```

**Purpose**: The specific rejection response used when the **Global Limit** is reached and the tenant has a bad reputation. Logs the specific reason for the ban (score, violations).

---

### **recover.go**

Provides crucial stability by preventing a panic in the middleware chain from crashing the application.

**Key Function:**

```go
func RecoveryMiddleware(next http.Handler, fallBack http.Handler, lgr *logger.Logger) http.Handler
```

**Function Logic:**

1.  **Defer/Recover**: Wraps the middleware chain in a `defer` block that calls `recover()`.
2.  **Panic Handling**: If a panic occurs, it is logged with the **stack trace**, and the request is immediately forwarded to the **backend target URL** (`fallBack` handler).
3.  **Fail-Open Strategy**: This implements the core fail-open strategy: any internal failure (including panic) will result in the request being successfully forwarded, ensuring high availability over strict enforcement.
4.  **Metrics**: Tracks `metrics.PanicRecoveries`.

---

## The Middleware Chain: Execution Flow

The middlewares are typically chained in this specific order to ensure correct execution and context setup:

1.  **`RecoveryMiddleware`**: Ensures `TrafficCTRL` remains highly available even in case of code panic (Fail-Open).
2.  **`MetadataMiddleware`**: Injects `X-Request-ID` and `ClientIP` into the request context.
3.  **`ClassifierMiddleware`**: Matches the request to a rate-limiting rule and extracts the `TenantKey`, setting up the request-scoped logger and the main context for all subsequent steps.
4.  **`DryRunMiddleware`** : Simulates all limit checks and logs the outcome without blocking traffic.
5.  **`GlobalLimitMiddleware`**: Checks for system-wide high load and bans bad-reputation tenants if load is exceeded.
6.  **`TenantLimitMiddleware`** (If enabled): Checks the overall limit for the specific tenant.
7.  **`EndpointLimitMiddleware`** : Checks the specific limit for the requested path/method. If allowed, this final middleware updates the tenant's reputation score (good request).
8.  **Target Proxy**: The request is forwarded to the main backend.
