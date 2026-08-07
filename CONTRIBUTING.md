# Contributing to GateShift

Thank you for helping improve Ingress → Gateway API migration safety.

## Development setup

```bash
git clone https://github.com/vi-shub/gateshift.git
cd gateshift
make tidy
make test
make build
```

## Project principles

1. **Never silently drop annotations.** Every migration-relevant key must produce an audit finding.
2. **L3 stays honest.** Do not invent Envoy/Lua equivalents without a tested pattern.
3. **Adapters over mega-switches.** New annotation families belong in `pkg/adapters/...`.
4. **Corpus grows with reality.** Customer/public fixtures land in `examples/corpus/` with tests.

## Adding an annotation adapter

1. Add the key constant in `pkg/adapters/nginx/keys.go` (or a new vendor package).
2. Implement `adapters.AnnotationAdapter` (`CanHandle` + `Transform`).
3. Register it in `DefaultAdapters()`.
4. Update `Catalog()` (`Implemented: true` when done).
5. Add table-driven unit tests.
6. Run:

```bash
go test ./pkg/adapters/...
gateshift coverage -f examples/your-fixture.yaml
gateshift audit -f examples/your-fixture.yaml --target=envoy-gateway
```

### Level guide

| Level | When to use |
|------:|-------------|
| L1 | Maps cleanly to Gateway API Core/Extended filters |
| L2 | Needs a provider Policy / Certificate CRD |
| L3 | Snippet, Lua, or no portable equivalent  -  flag + hints |

## Extending the snippet pattern library

Edit `pkg/patterns/snippet.go`:

- Match only idioms that are **fully accounted for** (or mark `Residual: true`).
- Prefer high-confidence promotions (`Confidence >= 0.8`) for `FullyCovered`.
- Always add a unit test in `pkg/patterns/snippet_test.go`.

## Adding corpus fixtures

1. Place YAML under `examples/corpus/`.
2. Document expected readiness / L3 in a short comment at the top of the file if non-obvious.
3. Prefer real shapes (canary pairs, ImplementationSpecific paths, mixed TLS).

## Cluster testing

Follow [docs/TESTING.md](docs/TESTING.md). On WSL, use the **Linux** binary (`bin/gateshift`), not `gateshift.exe`.

## Pull requests

- Keep PRs focused (one adapter family or one feature).
- Include tests for new behavior.
- Update `Catalog()` / docs when user-facing coverage changes.
- Do not commit `.gateshift-e2e/`, `.gateshift-pr/`, or kubeconfigs.

## Code style

- `gofmt` required
- Prefer small packages with clear boundaries (`adapters` → `ir` → `convert`)
- Avoid unrelated refactors in feature PRs

## License

Contributions are accepted under the Apache License 2.0.
