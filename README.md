# GateShift

Enterprise-grade migration suite for converting legacy Kubernetes Ingress resources (and NGINX annotations) into [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs — with honest L1/L2/L3 classification so production traffic does not silently break.

**CLI:** `gateshift` · **Operator:** `gateshift-operator` · **License:** Apache 2.0

## Design view (read this)

Annotation parsing is easy. **Translation difficulty depends on the annotation:**

| Level | Meaning | GateShift strategy |
|------:|---------|--------------------|
| **L1** | 1:1 Gateway API filters | Emit `URLRewrite`, `RequestRedirect`, header modifiers |
| **L2** | Needs controller Policy CRDs | Emit Envoy/Cilium/Istio/Kong policy stubs (`BackendTrafficPolicy`, `SecurityPolicy`, `Certificate`) |
| **L3** | Nginx magic (snippets/Lua) | Flag as untranslatable + actionable hints — never invent Envoy config |

Full rationale: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## Quick start

```powershell
make tidy
make test
make build

.\bin\gateshift.exe audit -f examples\ingress-checkout.yaml --target=envoy-gateway
.\bin\gateshift.exe convert -f examples\ingress-checkout.yaml --target=envoy-gateway -o gateway.yaml
.\bin\gateshift.exe validate -f examples\ingress-checkout.yaml --target=envoy-gateway
.\bin\gateshift.exe diff -f examples\ingress-with-snippet.yaml
.\bin\gateshift.exe migrate -f examples\ingress-checkout.yaml --target=envoy-gateway
```

Live cluster audit:

```powershell
.\bin\gateshift.exe audit --namespace shop --target=envoy-gateway
```

## CLI commands

| Command | Purpose |
|---------|---------|
| `audit` | L1/L2/L3 matrix (file or `--namespace`) |
| `convert` | Emit Gateway / HTTPRoute / Policy YAML |
| `diff` | Structural Ingress vs Gateway API comparison |
| `validate` | Controller capability matrix (conformance) |
| `migrate` | Convert + GitOps PR (or local dry-run artifacts) |

## Milestone status

| Milestone | Status |
|-----------|--------|
| M1 Annotation adapter engine (plugin registry) | Done |
| M2 CLI (`audit` / `convert` / `diff` / `validate` / `migrate`) | Done |
| M3 Conformance validation | Done (built-in provider profiles) |
| M4 Operator + `MigrationRequest` + GitOps | Done (scaffold + reconciler + Helm chart) |

## Repository layout

```
api/v1alpha1/           MigrationRequest types
cmd/gateshift/          CLI
cmd/gateshift-operator/ Operator manager
internal/cli/           Cobra commands
internal/controller/    MigrationRequest reconciler
pkg/adapters/           Plug-in AnnotationAdapter interface
pkg/adapters/nginx/     L1/L2/L3 NGINX adapters
pkg/ir/                 Intermediate representation
pkg/convert/            Ingress → IR → YAML
pkg/conformance/        GatewayClass capability checks
pkg/cluster/            Live kube Ingress listing
pkg/gitops/             GitHub PR + dry-run artifacts
pkg/audit/ pkg/diff/    Human reports
charts/gateshift-operator/  Helm chart
config/crd/             CRD manifests
docs/ARCHITECTURE.md    Design view
```

## Operator

```bash
kubectl apply -f config/crd/migrationrequest.yaml
# or: helm install gateshift charts/gateshift-operator
kubectl apply -f examples/migrationrequest.yaml
```

Build operator binary: `make build-operator`

## Supported adapters (initial set)

**L1:** rewrite-target, ssl-redirect, force-ssl-redirect, permanent/temporal-redirect, CORS headers  
**L2:** limit-rps/rpm/connections, cert-manager issuers, affinity, whitelist/denylist, proxy timeouts/body-size, canary, use-regex, auth-url→Envoy SecurityPolicy scaffold  
**L3:** configuration-snippet, server-snippet, modsecurity-snippet (+ regex hints for `more_set_headers` / UA denies)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
