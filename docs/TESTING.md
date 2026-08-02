# Testing GateShift (WSL + KinD)

You already have the right stack:

- **Docker Desktop** (WSL2 backend) — running
- **Ubuntu WSL** — running
- **kubectl** — installed
- **KinD** — installed by the e2e script if missing

## Fast path (recommended)

### 0) One-time: build a **Linux** binary (PowerShell)

WSL cannot run `gateshift.exe` (`Exec format error`).

```powershell
cd C:\Users\smsha\Desktop\GateShift
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
```

### 1) Smoke test on KinD (Ubuntu WSL)

```bash
cd /mnt/c/Users/smsha/Desktop/GateShift
export PATH=$HOME/bin:$PATH
bash scripts/test-smoke.sh
```

Expected: `PASS` and curl body `checkout-ok`.

> Do **not** use `curl http://127.0.0.1:8080` on KinD without MetalLB — LB EXTERNAL-IP stays `<pending>`. The smoke script uses `kubectl port-forward` instead.

What the script does:

1. Creates KinD cluster `gateshift` (ports `8080`/`8443` → node 80/443)
2. Installs Gateway API CRDs + Envoy Gateway
3. Runs `gateshift audit|convert|validate|coverage` on the checkout example
4. Deploys echo backends + applies Gateway/HTTPRoute
5. Prints `kubectl`/`curl` commands for traffic checks

### Useful flags

```bash
SKIP_CLUSTER=1 bash scripts/e2e-kind.sh   # reuse existing cluster
SKIP_EG=1 bash scripts/e2e-kind.sh        # CRDs only (faster, no data plane)
CLEANUP=1 bash scripts/e2e-kind.sh        # delete cluster when finished
```

## Manual step-by-step

### 1) Unit / offline tests (PowerShell)

```powershell
cd C:\Users\smsha\Desktop\GateShift
go test ./...
.\bin\gateshift.exe audit -f examples\ingress-checkout.yaml --target=envoy-gateway
.\bin\gateshift.exe convert -f examples\ingress-checkout.yaml --target=envoy-gateway -o gateway.yaml
.\bin\gateshift.exe coverage -f examples\ingress-checkout.yaml
```

### 2) Create cluster (Ubuntu WSL)

```bash
kind create cluster --name gateshift
kubectl cluster-info --context kind-gateshift
```

### 3) Install Gateway API + Envoy Gateway

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
kubectl apply -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml
kubectl wait -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available --timeout=5m
```

### 4) Convert + apply

```bash
./bin/gateshift.exe convert -f examples/ingress-checkout.yaml --target=envoy-gateway -o /tmp/gs.yaml
kubectl create ns shop
# apply backends (see scripts/e2e-kind.sh) then:
kubectl apply -f /tmp/gs.yaml   # Certificate/Policy may warn if CRDs missing — OK for smoke
kubectl get gateway,httproute -n shop -o wide
```

### 5) Live-cluster audit

```bash
kubectl apply -f examples/ingress-checkout.yaml
./bin/gateshift.exe audit --namespace shop --target=envoy-gateway
```

### 6) Traffic check

```bash
kubectl get gateway -n shop -o wide
# once ADDRESS is set (or via kind port-map):
curl -v -H 'Host: checkout.example.com' http://127.0.0.1:8080/api
```

## What “pass” means

| Layer | Pass criteria |
|-------|----------------|
| Unit tests | `go test ./...` green |
| Offline CLI | audit/convert/coverage produce expected L1/L2 matrix |
| Conformance | `validate` fails on snippet example; passes/warns on checkout |
| Cluster apply | Gateway + HTTPRoute Accepted by Envoy Gateway |
| Data plane | curl returns backend response for Host header |

## Cleanup

```bash
kind delete cluster --name gateshift
```
