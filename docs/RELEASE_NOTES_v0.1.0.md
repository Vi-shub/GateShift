# GateShift v0.1.0

First public release of **GateShift** — Ingress → Gateway API migration with annotation fidelity and dual-run cutover.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/vi-shub/GateShift/main/scripts/install.sh | bash
gateshift version
```

Or download the archive for your OS/arch from the Assets below, extract, and put `gateshift` on your `PATH`.

Build from source (Go 1.22+):

```bash
git clone https://github.com/vi-shub/GateShift.git
cd GateShift
make tidy test build
```

## Highlights

- **CLI:** `audit`, `convert`, `dual-run`, `diff`, `validate`, `migrate`, `coverage`, `scoreboard`
- **L1 / L2 / L3** annotation classification (NGINX + cert-manager catalog)
- **Typed IR** `gateshift.ir/v1` with finding IDs, fixes, and `RequiredFeatures`
- **Dual-run:** keep Ingress live; emit staging Gateway + `*-shadow` HTTPRoute
- **Ingress-NGINX quirks:** host-wide regex, trailing slash, optional `--preserve-nginx-regex` / `--emit-trailing-slash-redirects`
- **Multi-provider** targets: `envoy-gateway`, `cilium`, `istio`, `kong`, `standard`
- **Public corpus scoreboard** + KinD smoke (convert + dual-run)

## Quick start

```bash
gateshift audit -f examples/ingress-checkout.yaml --target=envoy-gateway
gateshift convert -f examples/ingress-checkout.yaml --target=envoy-gateway -o gateway.yaml
gateshift validate -f examples/ingress-checkout.yaml --target=envoy-gateway
gateshift dual-run -f examples/ingress-checkout.yaml --target=envoy-gateway -o dual-run.yaml
```

Recommended cutover: audit → convert/validate → dual-run (compare) → migrate/GitOps → flip DNS → delete Ingress last.

## Docs

- [README](https://github.com/vi-shub/GateShift#readme)
- [COMPARE](https://github.com/vi-shub/GateShift/blob/main/docs/COMPARE.md)
- [ROADMAP](https://github.com/vi-shub/GateShift/blob/main/docs/ROADMAP.md)
- [MIDDLE_LAYER](https://github.com/vi-shub/GateShift/blob/main/docs/MIDDLE_LAYER.md)
- [TESTING](https://github.com/vi-shub/GateShift/blob/main/docs/TESTING.md)

## Notes

- Operator / Helm chart are **scaffold** in this release (CLI is the primary surface).
- Snippets / Lua remain L3 by design (reported, not silently dropped).
- Windows arm64 CLI archive is not published (see GoReleaser ignore rules).

## Checksums

See `checksums.txt` in Assets.
