# Ingress-NGINX behavioral quirk fixtures

Scenarios from the Kubernetes blog:

[Before You Migrate: Five Surprising Ingress-NGINX Behaviors You Need to Know](https://kubernetes.io/blog/2026/02/27/ingress-nginx-before-you-migrate/)
(Steven Jin, 2026-02-27)

| File | Quirk |
|------|-------|
| `01-regex-case-insensitive.yaml` | Regex = prefix + case-insensitive |
| `02-regex-host-wide.yaml` | `use-regex` applies host-wide across Ingresses |
| `03-rewrite-implies-regex.yaml` | `rewrite-target` implies regex |
| `04-trailing-slash.yaml` | `/path` → 301 `/path/` |
| `05-url-normalization.yaml` | `.` / `..` / `//` normalization |

```bash
gateshift audit -f examples/corpus/blog-k8s-2026-02/02-regex-host-wide.yaml --target=envoy-gateway
gateshift convert -f examples/corpus/blog-k8s-2026-02/02-regex-host-wide.yaml \
  --target=envoy-gateway --preserve-nginx-regex -o out.yaml
gateshift convert -f examples/corpus/blog-k8s-2026-02/04-trailing-slash.yaml \
  --target=envoy-gateway --emit-trailing-slash-redirects -o out.yaml
```

See [docs/BEHAVIORAL_FIDELITY.md](../../../docs/BEHAVIORAL_FIDELITY.md).
