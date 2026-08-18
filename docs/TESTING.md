# Testing Guide

This document describes how to verify GateShift locally and on a KinD cluster.

## Test layers

| Layer | Command / tool | Pass criteria |
|-------|----------------|---------------|
| Unit | `make test` / `go test ./...` | All packages green |
| IR goldens / invariants | `go test ./pkg/convert/ -update-goldens` | Stable `gateshift.ir/v1` JSON |
| Fuzz middle layer | `go test ./pkg/convert/ -fuzz=FuzzFromIngresses -fuzztime=30s` | No panic; schema + finding IDs |
| Offline CLI | `audit` / `convert` / `validate` / `coverage` | Expected L1/L2/L3 outcomes |
| Corpus scoreboard | `make scoreboard` / `gateshift scoreboard` | All providers scored; unreported annotations = 0 |
| Conformance gate | `validate` on snippet fixtures | Must **FAIL** on hard L3 |
| Cluster smoke (convert) | `scripts/test-smoke.sh` (CI: `.github/workflows/smoke.yml`) | Envoy returns backend body (`checkout-ok`) |
| Dual-run KinD e2e | `scripts/test-dual-run.sh` (CI: dual-run smoke job) | Shadow applied; Ingress uid unchanged; staging curl OK |
| Helm chart | `helm lint` / `make helm-lint` (CI: `helm-lint` job) | Chart lints and templates |
| Release | `.github/workflows/release.yml` + GoReleaser | Tagged `v*` publishes multi-OS binaries |

### Coverage gaps still open

Still thin or missing: `internal/cli`, `internal/controller`, `pkg/gitops`, `pkg/diff`. Full list: [ROADMAP.md §4](ROADMAP.md).

## Offline CLI (any OS)

```bash
make test
make build

gateshift audit -f examples/ingress-checkout.yaml --target=envoy-gateway
gateshift convert -f examples/ingress-checkout.yaml --target=envoy-gateway -o gateway.yaml
gateshift validate -f examples/ingress-checkout.yaml --target=envoy-gateway
gateshift coverage -f examples/ingress-checkout.yaml

# Multi-provider corpus scoreboard
make scoreboard
# or: gateshift scoreboard -f examples/corpus -o docs/scoreboard.md

# Expected FAIL (L3 block):
gateshift validate -f examples/ingress-with-snippet.yaml --target=envoy-gateway
```

Windows PowerShell uses `.\bin\gateshift.exe` instead of `gateshift`.

## Linux binary for WSL

WSL cannot execute `gateshift.exe` (`Exec format error`). Cross-compile from Windows:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
```

## KinD smoke test (recommended)

Prerequisites: Docker Desktop, Ubuntu WSL, `kubectl`, `kind` (`~/bin` is fine).

```bash
cd /mnt/c/Users/<user>/Desktop/GateShift   # adjust path
export PATH=$HOME/bin:$PATH
bash scripts/test-smoke.sh
```

The script will:

1. Apply Envoy Gateway with `kubectl apply --server-side` (avoids CRD annotation size limits)
2. Ensure `GatewayClass/envoy` exists
3. Create a dummy TLS secret for HTTPS listeners
4. Run `gateshift convert` and apply Gateway + HTTPRoute
5. `kubectl port-forward` the Envoy proxy Service (KinD LoadBalancers stay `<pending>`)
6. `curl` with `Host: checkout.example.com`

Expected output includes: `PASS` and body `checkout-ok`.

## Dual-run KinD e2e

Proves the shadow cutover path: Ingress stays live while a staging Gateway + `*-shadow` HTTPRoute serve traffic.

```bash
cd /mnt/c/Users/<user>/Desktop/GateShift   # adjust path
export PATH=$HOME/bin:$PATH
bash scripts/test-dual-run.sh
```

The script will:

1. Ensure Envoy Gateway + `GatewayClass/envoy`
2. Apply checkout backends **and** the live Ingress
3. Record Ingress uid, then run `gateshift dual-run`
4. Assert YAML has dual-run annotations / staging names and **no** Ingress docs
5. Apply Gateway + HTTPRoute only
6. Assert Ingress uid unchanged and `checkout-shadow` annotated `gateshift.io/mode=dual-run`
7. Port-forward staging Envoy Service and curl `Host: checkout.example.com`

Expected: `PASS - dual-run applied shadow path; Ingress left live`.

CI runs this as the **dual-run smoke** job in `.github/workflows/smoke.yml`.

### Manual dual-run on any cluster

```bash
gateshift dual-run -f examples/ingress-checkout.yaml --target=envoy-gateway -o dual-run.yaml
kubectl apply -f dual-run.yaml
kubectl get ingress,gateway,httproute -n shop
```

### Helm chart lint

```bash
make helm-lint
```

### Manual traffic check

```bash
kubectl -n envoy-gateway-system port-forward svc/envoy-shop-checkout-gateway-a55d33c9 18080:80
curl -H 'Host: checkout.example.com' http://127.0.0.1:18080/api
```

### Live Ingress audit

`audit --namespace` (or `--all-namespaces` / `--selector`) requires Ingress objects in-cluster (Gateway/HTTPRoute alone is not enough):

```bash
kubectl apply -f examples/ingress-checkout.yaml
gateshift audit --namespace shop --target=envoy-gateway

# Fleet batch: per-Ingress score table + combined matrix
gateshift audit --all-namespaces --selector app=checkout --target=envoy-gateway
```

Lab convert without TLS secrets:

```bash
gateshift convert -f examples/ingress-checkout.yaml --target=envoy-gateway --http-only -o gateway-http.yaml
gateshift dual-run --namespace shop --http-only --target=envoy-gateway -o dual-run.yaml
```

### Full e2e script

```bash
# Reuse cluster, install Envoy Gateway, run suite
SKIP_CLUSTER=1 bash scripts/e2e-kind.sh

# Faster CRD-only path
SKIP_CLUSTER=1 SKIP_EG=1 bash scripts/e2e-kind.sh
```

### Cleanup

```bash
kind delete cluster --name gateshift
```

## Expanding coverage

1. Add fixtures under `examples/corpus/`
2. Run `gateshift coverage -f <file>` and note `[GAP]` / `[??]`
3. Implement adapter or pattern (see [CONTRIBUTING.md](../CONTRIBUTING.md))
4. Add unit tests; re-run smoke for routing-sensitive changes

## Artifacts

| Path | Purpose |
|------|---------|
| `.gateshift-e2e/` | Temp convert/apply YAML from smoke/e2e scripts |
| `.gateshift-pr/` | Dry-run PR pack from `gateshift migrate` |

Both are gitignored and safe to delete.
