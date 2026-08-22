# GateShift v0.1.1

Hardening release focused on **Envoy Gateway apply compatibility**.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vi-shub/GateShift/main/scripts/install.sh | bash
gateshift version
```

Or download the `v0.1.1` assets from the GitHub Release.

## Fixes

- **BackendTrafficPolicy** emission matches Envoy Gateway CRDs (EG 1.2+ portable field set)
  - Cookie affinity → `loadBalancer.consistentHash`
  - Timeouts / body → `timeout.http` / `timeout.tcp` / `connection.bufferLimit`
  - Targets → `targetRefs` without `namespace`
- Emit-time **allowlist sanitizer** drops unknown / newer-only / IR-only keys so
  `kubectl apply` does not fail on strict decoding
- Skips incomplete scaffolds that would fail apply (`extAuth` without backends,
  invalid `ClientTrafficPolicy` / UA-deny SecurityPolicy shells)
- Cleaner Gateway/HTTPRoute YAML (no empty `status` / `creationTimestamp`)
- Fleet flags and `--http-only` from the prior sprint remain available

## Docs

- [Envoy Gateway compatibility](EG_COMPAT.md)

## Quick verify

```bash
gateshift dual-run -f examples/demo-podinfo/02-ingress.yaml --target=envoy-gateway -o dual-run.yaml
kubectl apply --dry-run=server -f dual-run.yaml
kubectl apply -f dual-run.yaml
```
