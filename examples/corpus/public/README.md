# Public corpus fixtures

Curated Ingress shapes used by `gateshift scoreboard` to compare providers
(`standard`, `envoy-gateway`, `cilium`, `istio`, `kong`) and measure
annotation reporting versus a structure-only baseline.

| # | File | Focus |
|---|------|-------|
| 01 | `01-basic-hosts-paths.yaml` | Structure only |
| 02 | `02-rewrite-ssl-redirect.yaml` | L1 rewrite + redirect |
| 03 | `03-ratelimit-ipallow.yaml` | L2 rate limit + IP allow |
| 04 | `04-cors-cert-manager.yaml` | CORS + Certificate |
| 05 | `05-affinity-session.yaml` | Session cookie affinity |
| 06 | `06-proxy-timeouts-body.yaml` | Proxy timeouts / body size |
| 07 | `07-canary-pair.yaml` | Canary weight merge |
| 08 | `08-mirror-regex.yaml` | Mirror + regex path |
| 09 | `09-auth-url.yaml` | External auth scaffold |
| 10 | `10-snippet-modsecurity.yaml` | L3 snippets / WAF |
| 11 | `11-ssl-passthrough-default-backend.yaml` | TLS passthrough |
| 12 | `12-backend-tls-upstream-hash.yaml` | Backend TLS + hash |
| 13 | `13-permanent-temporal-redirect.yaml` | Permanent + www redirect |
| 14 | `14-denylist-modsecurity.yaml` | IP deny + ModSecurity L3 |
| 15 | `15-app-root-www-redirect.yaml` | App-root + www + SSL redirect |

Also see `examples/corpus/github/`, `traefik/`, and `community/` for upstream fixtures.

```bash
gateshift scoreboard -f examples/corpus -o docs/scoreboard.md
```
