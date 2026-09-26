# Troubleshooting

Common errors and fixes when using GateShift.

---

## "no matches for kind BackendTrafficPolicy"

**Error:**
```
resource mapping not found for name: "podinfo-affinity" namespace: "podinfo" from "dual-run.yaml": 
no matches for kind "BackendTrafficPolicy" in version "gateway.envoyproxy.io/v1alpha1"
ensure CRDs are installed first
```

**Cause:** Envoy Gateway controller and its extension CRDs are not installed on your cluster. GateShift generates `BackendTrafficPolicy` resources for session affinity, rate limiting, timeouts, etc. when targeting `--target=envoy-gateway`.

**Fix options:**

### Option 1: Install Envoy Gateway (recommended for production)

```bash
kubectl apply --server-side --force-conflicts -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml
```

Or use the helper script:
```bash
bash scripts/ensure-envoy-gateway.sh
```

Wait ~30 seconds, then verify:
```bash
kubectl get crd backendtrafficpolicies.gateway.envoyproxy.io
```

### Option 2: Skip extension policies (quick workaround)

If you only need Gateway + HTTPRoute without BackendTrafficPolicy/SecurityPolicy:

```bash
gateshift dual-run -f ingress.yaml --target=envoy-gateway --skip-extension-policies -o dual-run.yaml
```

This emits only core Gateway API resources (Gateway, HTTPRoute) without provider-specific policy CRDs.

### Option 3: Use `--target=standard`

For pure Gateway API without any provider extensions:

```bash
gateshift convert -f ingress.yaml --target=standard -o gateway.yaml
```

---

## "kubeconfig ... no such file or directory"

**Error:**
```
Error: kubeconfig (stat /root/.kube/config: no such file or directory) and in-cluster ... failed
```

**Cause:** GateShift couldn't find your kubeconfig file. On K3s, the config is at `/etc/rancher/k3s/k3s.yaml`, not `~/.kube/config`.

**Fix options:**

### Option 1: Set KUBECONFIG environment variable

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml  # K3s
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml  # RKE2
```

### Option 2: Copy to standard location

```bash
mkdir -p ~/.kube
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
chmod 600 ~/.kube/config
```

### Option 3: Use --kubeconfig flag

```bash
gateshift audit --namespace podinfo --kubeconfig=/etc/rancher/k3s/k3s.yaml
```

---

## Preflight check

Run preflight to verify all prerequisites before migration:

```bash
gateshift preflight --target=envoy-gateway
```

Expected output when everything is ready:
```
GateShift Preflight Check (target: envoy-gateway)
=====================================

✅ Gateway
✅ HTTPRoute
✅ GatewayClass
✅ BackendTrafficPolicy
✅ SecurityPolicy

✅ All preflight checks passed!
```

If checks fail, use `--fix` to see installation commands:
```bash
gateshift preflight --target=envoy-gateway --fix
```

---

## "unknown field" errors during kubectl apply

**Error:**
```
error: error validating "dual-run.yaml": error validating data: 
ValidationError(BackendTrafficPolicy.spec): unknown field "someField"
```

**Cause:** The generated policy contains fields not supported by your Envoy Gateway version.

**Fix:** GateShift sanitizes policies for EG 1.2+ compatibility. Ensure you have Envoy Gateway v1.2.0 or later:

```bash
kubectl apply --server-side --force-conflicts -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml
```

Or use `--skip-extension-policies` to avoid emitting these resources entirely.

---

## Gateway/HTTPRoute work but BackendTrafficPolicy fails

This is the same issue as "no matches for kind BackendTrafficPolicy" above. Gateway API core CRDs (Gateway, HTTPRoute, GatewayClass) are separate from Envoy Gateway extension CRDs.

**Quick check:**
```bash
# Core Gateway API CRDs (usually present)
kubectl get crd gateways.gateway.networking.k8s.io

# Envoy Gateway extension CRDs (require EG install)
kubectl get crd backendtrafficpolicies.gateway.envoyproxy.io
```

If the second command fails, install Envoy Gateway or use `--skip-extension-policies`.
