# Envoy Gateway compatibility

GateShift `--target=envoy-gateway` emits Gateway API core resources plus
`gateway.envoyproxy.io/v1alpha1` policies that are intended to **`kubectl apply`
cleanly on Envoy Gateway 1.2 and newer**.

## Supported baseline

| Surface | Compatibility approach |
|---------|------------------------|
| `Gateway` / `HTTPRoute` | Gateway API v1 (portable) |
| `BackendTrafficPolicy` | Only fields stable since **EG 1.2** |
| `SecurityPolicy` | `authorization`, `basicAuth`, `cors`, `jwt`, `oidc`, `extAuth` (when complete) |
| Targeting | `spec.targetRefs` (no `namespace`; same-namespace LocalPolicyTargetReference) |

## Explicitly not emitted (by design)

These either invent non-CRD fields or appear only on newer EG builds, so GateShift
keeps them as **audit findings** instead of apply-breaking YAML:

- Fake IR leftovers (`affinity`, `maxRequestBodySize`, `readTimeout`, `featureGate`, …)
- Newer-only BTP fields (`requestBuffer`, `bandwidthLimit`, `admissionControl`, BTP `telemetry`, …)
- Incomplete scaffolds (`extAuth` without `backendRefs`, UA-deny SecurityPolicy shells)
- `ClientTrafficPolicy` (listener/client schema varies; prefer EnvoyProxy / manual)

## Cookie affinity

Ingress-NGINX `affinity=cookie` maps to:

```yaml
spec:
  loadBalancer:
    type: ConsistentHash
    consistentHash:
      type: Cookie
      cookie:
        name: <session-cookie-name>
        ttl: <max-age>s
```

This is valid on EG BackendTrafficPolicy across 1.2+. Route-level Gateway API
`sessionPersistence` is **not** used for emission today (controller support varies).

## Timeouts / body size

| Ingress annotation | EG field (portable) |
|--------------------|---------------------|
| `proxy-read-timeout` | `timeout.http.requestTimeout` |
| `proxy-connect-timeout` | `timeout.tcp.connectTimeout` |
| `proxy-body-size` | `connection.bufferLimit` (Quantity, e.g. `8Mi`) |

## Verify on your cluster

```bash
gateshift dual-run -f ingress.yaml --target=envoy-gateway -o dual-run.yaml
kubectl apply --dry-run=server -f dual-run.yaml
kubectl apply -f dual-run.yaml
```

If dry-run fails on a policy field, open an issue with your **Envoy Gateway version**
(`envoy-gateway version` / chart appVersion) and the rejected YAML snippet.
