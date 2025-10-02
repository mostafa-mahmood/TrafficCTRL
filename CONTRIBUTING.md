# Contributing to TrafficCTRL

Whether you want to fix a minor bug, enhance an existing feature, write robust tests or develop a completely new feature, your help is incredibly valuable. Every contribution shapes the future of this project.

---

## What to Contribute

We are open to any change that improves the performance, reliability, and functionality of TrafficCTRL.

| Contribution Type       | Focus Areas                                                                                                                                                                                                                                                                              |
| :---------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Reporting Bugs**      | If you find an issue, please help us diagnose it by opening a new issue with: **Steps to Reproduce:** A clear, concise sequence of steps to replicate the bug, **Context:** Include your Go version, Redis version, and relevant configuration files (e.g., portions of `limiter.yaml`). |
| **Fixes & Refactoring** | Optimizing our **Redis Lua scripts** for the rate-limiting core, improving error handling (especially around Redis connectivity), and general **Go idiomatic refactoring** for clarity and speed.                                                                                        |
| **Enhancements**        | Improving the **Reputation System** logic, adding more flexible **Tenant Key strategies** (e.g., custom regex matching), or enhancing our existing **Prometheus metrics** and **Zap structured logs**.                                                                                   |
| **New Features**        | adding new admission control layers (check [ROADMAP.md](./ROADMAP.md)), or extending configuration capabilities.                                                                                                                                                                         |

---

## Getting Started

### Local Development Setup

1.  **Prerequisites:** You need **Go 1.20+** and a running instance of **Redis 7+**.
2.  **Clone the Repository:**
    ```bash
    git clone https://github.com/mostafa-mahmood/TrafficCTRL.git
    cd TrafficCTRL
    ```
3.  **Run Dependencies (Easiest with Docker):**
    For a consistent environment, you can use `docker-compose` to spin up Redis and a mock backend, or simply start Redis locally.
4.  **Run TrafficCTRL:**
    ```bash
    go mod tidy
    go run cmd/ctrl/main.go
    ```
    _(Note: This requires valid configuration files in your `/config` directory, as outlined in the main README.)_

## Contribution Workflow

We use a standard Fork & Pull Request model.

1.  **Fork** the repository to your own GitHub account.
2.  **Clone** your fork locally and create a new branch:
    ```bash
    git checkout -b fix/clear-bug-in-reputation-logic
    # OR
    git checkout -b feat/add-new-endpoint-strategy
    ```
3.  **Commit** your changes. Please use clear, descriptive commit messages, ideally following a conventional commit style (e.g., `fix(packageName):`, `feat(packageName):`, `docs(packageName):`).
4.  **Push** your branch and **open a Pull Request (PR)** against the `main` branch of the original repository.

## Quality Guidelines

**Main goal is to Focus on enhancing it without breaking what already works**

### Code Style and Quality

- **Go Idioms:** Follow standard, idiomatic Go. Run `go fmt` and `go vet` before committing.
- **Logging:** All new logic, especially error paths, **must** use the **Zap structured logger** provided via context. Log messages should be clear and include relevant fields like `tenant_key` or `limit_level`.
- **Observability:** If your contribution introduces new rejection logic or affects system-wide performance, ensure it is exposed via the **Prometheus metrics**. Specifically, update the `DeniedRequests` or similar metrics with appropriate labels (e.g., `global`, `tenant`, `endpoint`).

### Documentation Standards

If your code changes touch any of the core packages, the corresponding documentation files in the `docs/` folder **must** be updated.

- [limiter_package.md](./docs/limiter_package.md) (for changes to algorithms or Redis logic)
- [middleware_package.md](./docs/middleware_package.md) (for changes to the request processing chain)
- [config_package.md](./docs/config_package.md) (for changes to configuration)
- [proxy_package.md](./docs/proxy_package.md)
- [file_structure.md](./docs/file_structure.md)

---

Thank you for helping, this tool will be from the community and to the community.
