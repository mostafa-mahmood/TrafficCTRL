# TrafficCTRL Roadmap

TrafficCTRL is just getting started — here’s where we’re heading next.  
This document outlines planned features, improvements, and stretch goals for the project.

---

## Current Capabilities

- Multi-algorithm rate limiting (Token Bucket, Leaky Bucket, Fixed Window, Sliding Log)
- Layered admission control (Global, Per-Tenant, Per-Endpoint)
- Reputation system (anti-abuse / progressive penalties)
- Flexible tenant keys (headers, cookies, query params, IPs)
- Dry run mode
- Observability (Prometheus metrics + structured logging)

## Short Term (Next Releases)

**Core admission control + quality-of-life features**

- [ ] **IP / CIDR Whitelists & Blacklists** (manual or external feed integration)
- [ ] **Geo-based Access Control** (allow/block by region or ASN)
- [ ] **Time-based Rules** (configure per-hour/day/week limits)
- [ ] **Delays for Abusers (Tarpitting)** → slow down instead of instantly blocking
- [ ] **Hot Reload Config** → apply config changes without restart
- [ ] **Alerting Hooks** → send notifications via Webhooks, Slack, or Discord

## Mid Term

- [ ] **Adaptive Reputation System** → reputation informed by traffic patterns, not just rate-limit hits
- [ ] **Progressive Penalties 2.0** → delay → temp ban → permanent ban escalation
- [ ] **Bot Fingerprinting** → detect malicious user-agents, header anomalies, or automation patterns
- [ ] **Challenge Mode** → optional proof-of-work or external CAPTCHA integration
- [ ] **Per-Tenant Dashboards** (Grafana-ready views for usage/violations/reputation)
- [ ] **Audit Trails** → store tenant violation history for forensic analysis
- [ ] **Plugin / Policy Scripts** (Lua/JS) → let users define custom admission logic

## Long Term (Vision)

- [ ] **Tenant Quotas** (daily/monthly limits, like API monetization)
- [ ] **Integration with API Keys / JWTs** as tenant identifiers
- [ ] **Multi-Backend Failover** → optional fallback backend if target service is down
- [ ] **Inline Sanitization (Basic WAF-lite)** → reject malformed or suspicious requests before backend
- [ ] **Machine Learning-based Anomaly Detection** → auto-adjust limits based on historical baselines
- [ ] **Enterprise Integrations** → plug into SIEMs, threat intel feeds, and cloud-native monitoring tools

## How to Contribute

If you’d like to jump in:

- See [CONTRIBUTING.md](./CONTRIBUTING.md)
- Open issues for feature requests, bugs, or proposals
- Submit PRs aligned with this roadmap

---

> TrafficCTRL’s goal: stay **lightweight, fast, and focused** — admission control without bloat.  
> This roadmap is ambitious, but we’ll grow carefully to keep that philosophy intact.
