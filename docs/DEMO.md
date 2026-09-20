# GateShift demo & recording guide

**Story (in this order):** normal podinfo Ingress → install from GitHub **release** → walk major commands → **dual-run apply** → prove Ingress still live.

| Skip on camera | Why |
|----------------|-----|
| `gateshift scoreboard` | Slow; not the product story |
| `scripts/test-dual-run.sh` | CI smoke only |
| Operator / Helm | Scaffold, not the hero |

Envoy Gateway apply notes: [EG_COMPAT.md](EG_COMPAT.md)

---

## What is remaining before you shoot

### Already done (product)

- [x] `v0.1.1` published: https://github.com/Vi-shub/GateShift/releases/tag/v0.1.1  
- [x] Install script + Linux/macOS/Windows assets  
- [x] Dual-run YAML that applies on Envoy Gateway (policies sanitized)  
- [x] Demo manifests: `examples/demo-podinfo/`  
- [x] This guide + `scripts/demo-podinfo.sh`

### You still do (filming blockers)

Do these **once offline**, then hit record.

| # | Task | Pass when |
|---|------|-----------|
| 1 | Cluster warm | KinD (or any) up; Envoy Gateway Ready; `kubectl get gatewayclass envoy` works |
| 2 | Release install in WSL | `curl …/install.sh \| bash` then `gateshift version` prints **0.1.1** (Linux binary, not `.exe`) |
| 3 | Full rehearsal | Run [Pre-flight rehearsal](#pre-flight-rehearsal-run-once-offline) end-to-end with **zero** apply errors |
| 4 | Terminal look | Font ≥16–18pt; hide secrets; one clear prompt; zoom ~125–150% |
| 5 | Optional B-roll | 10s GitHub Releases page + repo README open in browser |

### Nice-to-have (not blocking)

- [ ] LinkedIn/YouTube thumbnail (repo logo + “dual-run”)  
- [ ] Captions / chapters after edit  
- [ ] Face cam intro (optional)

When **1–4** are green, you are ready to shoot.

---

## Length & chapters

**Target: 8–10 minutes** (hard cap 12).

| Time | Chapter title |
|------|----------------|
| 0:00 | Why GateShift |
| 0:25 | Normal app + Ingress |
| 1:30 | Install from release |
| 2:00 | Audit / coverage / diff / validate |
| 4:30 | Convert vs dual-run |
| 6:00 | Apply shadow path |
| 7:30 | Prove Ingress still live (+ optional curl) |
| 8:30 | Close + links |

---

## Prerequisites

| Item | Notes |
|------|--------|
| Cluster | KinD `kind-gateshift` (or any) + Envoy Gateway + `GatewayClass/envoy` |
| Shell | Ubuntu WSL |
| Binary | **Release** install preferred (`v0.1.1`) |
| Repo clone | For `examples/demo-podinfo` manifests |

```bash
curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash
export PATH="$HOME/bin:$PATH"
gateshift version   # expect 0.1.1
```

Go module (10 seconds on screen only):

```text
module github.com/gateshift/gateshift
# embed pkg/convert, pkg/ir in your own tools / operator
```

---

## Pre-flight rehearsal (run once offline)

```bash
cd /mnt/c/Users/<you>/Desktop/GateShift   # adjust path
export PATH="$HOME/bin:$PATH"

# Cluster + Envoy Gateway already installed...
kubectl get gatewayclass envoy
kubectl get deploy -n envoy-gateway-system

# Clean slate for podinfo (optional)
kubectl delete ns podinfo --ignore-not-found

# Full path (or: bash scripts/demo-podinfo.sh)
kubectl apply -f examples/demo-podinfo/01-app.yaml
kubectl apply -f examples/demo-podinfo/02-ingress.yaml
kubectl -n podinfo rollout status deploy/podinfo --timeout=180s

mkdir -p /tmp/gs-demo
kubectl -n podinfo get ingress podinfo -o yaml > /tmp/gs-demo/ingress.yaml

gateshift audit -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway
gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/dual-run.yaml

kubectl apply --dry-run=server -f /tmp/gs-demo/dual-run.yaml
kubectl apply -f /tmp/gs-demo/dual-run.yaml

kubectl -n podinfo get ingress,gateway,httproute,backendtrafficpolicy
```

**Pass criteria**

- `dry-run=server` and `apply` succeed with **no** `unknown field` errors  
- Ingress `podinfo` still exists  
- You see staging Gateway + `podinfo-shadow` HTTPRoute (+ BackendTrafficPolicy docs)  

If apply fails, fix offline — do not film that take.

---

## Shot list + voiceover

### 0. Open (20s)

**Show:** GitHub repo → Releases `v0.1.1`.

**Say:**  
“Teams still run Ingress. Gateway API is the destination. GateShift audits annotation fidelity and dual-runs a shadow path so cutover is safe. Install from a GitHub release — or import the Go module if you embed it.”

### 1. Setup podinfo — no GateShift yet (60–90s)

```bash
kubectl apply -f examples/demo-podinfo/01-app.yaml
kubectl apply -f examples/demo-podinfo/02-ingress.yaml
kubectl -n podinfo get pods,svc,ingress
```

Open `02-ingress.yaml` briefly: rewrite, CORS, cookie affinity, body size, timeouts.

**Say:**  
“Normal Kubernetes app, normal NGINX-style Ingress. GateShift is not installed yet.”

### 2. Install GateShift (30s)

```bash
curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash
export PATH="$HOME/bin:$PATH"
gateshift version
```

**Say:**  
“Same one-liner end users get from the release.”

### 3. Major commands (4–6 min)

```bash
mkdir -p /tmp/gs-demo
kubectl -n podinfo get ingress podinfo -o yaml > /tmp/gs-demo/ingress.yaml
```

| Beat | Command | Say |
|------|---------|-----|
| Audit (file) | `gateshift audit -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway` | “L1 direct, L2 needs a policy CRD, L3 needs a human. Honest score.” |
| Audit (live) | `gateshift audit --namespace podinfo --target=envoy-gateway` | “Same matrix against the live cluster.” |
| Coverage | `gateshift coverage -f /tmp/gs-demo/ingress.yaml` | “Which annotations we catalog vs gaps.” |
| Diff | `gateshift diff -f /tmp/gs-demo/ingress.yaml` | “Structural Ingress vs Gateway shape.” |
| Validate | `gateshift validate -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway` | “Fail closed if the target cannot support a feature.” |
| Convert | `gateshift convert -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/gateway.yaml` | “Full rewrite YAML — fine for greenfield; riskier alone in production.” |
| Dual-run | `gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/dual-run.yaml` | **Hero:** “Staging Gateway + shadow HTTPRoute. Ingress is never modified.” |
| Migrate | `gateshift migrate -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway` | “GitOps dry-run under `.gateshift-pr/`.” |

**Pause** on the audit matrix and the dual-run checklist (stderr).

Optional one-liner for labs without certs (only if you mention TLS pain):

```bash
gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway --http-only -o /tmp/gs-demo/dual-run-http.yaml
```

### 4. Incorporate changes (2 min)

```bash
kubectl apply --dry-run=server -f /tmp/gs-demo/dual-run.yaml
kubectl apply -f /tmp/gs-demo/dual-run.yaml

kubectl -n podinfo get ingress podinfo
kubectl -n podinfo get gateway,httproute,backendtrafficpolicy
```

**Say:**  
“Ingress is still live. We added a parallel Gateway API path. Flip DNS when confident; delete Ingress last.”

Optional curl:

```bash
# Discover Envoy proxy Service for the staging Gateway, then:
kubectl get svc -n envoy-gateway-system | grep -i podinfo || true
# kubectl -n envoy-gateway-system port-forward svc/<envoy-svc> 18081:80
curl -sS -H 'Host: podinfo.local' http://127.0.0.1:18081/ | head
```

If curl is flaky on camera, skip it — the `kubectl get` payoff is enough.

### 5. Close (20s)

**Say:**  
“Audit → validate → dual-run → apply → cut over. GateShift v0.1.1 — install link and repo in the description.”

---

## Exact copy-paste block (filming terminal)

Keep this in a second window; paste beat-by-beat (do not dump all at once).

```bash
export PATH="$HOME/bin:$PATH"
cd /mnt/c/Users/<you>/Desktop/GateShift

kubectl apply -f examples/demo-podinfo/01-app.yaml
kubectl apply -f examples/demo-podinfo/02-ingress.yaml
kubectl -n podinfo get pods,svc,ingress

gateshift version

mkdir -p /tmp/gs-demo
kubectl -n podinfo get ingress podinfo -o yaml > /tmp/gs-demo/ingress.yaml

gateshift audit -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway
gateshift audit --namespace podinfo --target=envoy-gateway
gateshift coverage -f /tmp/gs-demo/ingress.yaml
gateshift diff -f /tmp/gs-demo/ingress.yaml
gateshift validate -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway
gateshift convert -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/gateway.yaml
gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/dual-run.yaml
gateshift migrate -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway

kubectl apply --dry-run=server -f /tmp/gs-demo/dual-run.yaml
kubectl apply -f /tmp/gs-demo/dual-run.yaml
kubectl -n podinfo get ingress,gateway,httproute,backendtrafficpolicy
```

---

## Recording tips

- One command → pause 1s → talk → next command  
- Do not scroll long YAML; show audit matrix + dual-run checklist + `kubectl get`  
- If a command hangs, cut in edit; keep a second terminal with `kubectl get` ready  
- Prefer **one continuous terminal take**; add intro/outro separately  

---

## Description blurb (paste under video)

```text
GateShift: Kubernetes Ingress → Gateway API with L1/L2/L3 annotation reporting and dual-run (shadow) cutover.

Repo: https://github.com/Vi-shub/GateShift
Release: https://github.com/Vi-shub/GateShift/releases/tag/v0.1.1
Install: curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash

Chapters:
0:00 Why GateShift
0:25 Normal app + Ingress
1:30 Install from release
2:00 Audit / coverage / diff / validate
4:30 Convert vs dual-run
6:00 Apply shadow path
7:30 Prove Ingress still live
8:30 Close
```

### LinkedIn short caption

```text
Ingress → Gateway API without a blind rewrite.

GateShift v0.1.1: audit annotations (L1/L2/L3), then dual-run a staging Gateway + shadow HTTPRoute while Ingress stays live.

Install: curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash
Repo: https://github.com/Vi-shub/GateShift
```

---

## Automated helper

```bash
bash scripts/demo-podinfo.sh
```

Same dual-run path for rehearsal. Needs Linux `gateshift` on `PATH` and a usable kube context.

CI only (not for video): `bash scripts/test-dual-run.sh`
