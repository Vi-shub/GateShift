# Security Policy

## Supported versions

Security fixes are applied on the default branch and the latest release tag.

## Reporting a vulnerability

Please report security issues privately via the repository’s security advisory workflow (GitHub Security Advisories) or the maintainer contact listed on the GitHub repository.

Include:

- Affected component (`gateshift` CLI, operator, Helm chart)
- GateShift / Kubernetes / Gateway controller versions
- Reproduction steps and impact assessment

Do **not** open a public issue for unpatched vulnerabilities.

## Scope notes

- GateShift generates Kubernetes manifests and may open GitOps PRs when tokens are provided. Treat `GITHUB_TOKEN` and kubeconfig credentials as secrets.
- Converted YAML should be reviewed before production apply; L2 Policy CRDs and L3 findings require human ownership.
- The operator’s dry-run defaults should remain enabled until GitOps automation is intentionally trusted in the target environment.
