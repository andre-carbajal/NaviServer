# NaviServer project documentation

This documentation baseline is for **maintainers and developers**. It describes current implementation alongside explicitly labeled owner-approved behavior. The confirmed deployment scope is personal and small self-hosted installations on trusted networks (LAN or VPN). It is not a roadmap or a promise of future behavior.

## Areas

| Area | Purpose | Index |
|---|---|---|
| Context | Product purpose, users, scope, assumptions, and open decisions. | [00-context](00-context/index.md) |
| Functionality | Current capabilities, requirements, rules, stories, use cases, and traceability. | [01-functional](01-functional/index.md) |
| Architecture | Current system boundary, structure, data, integrations, and deployment shape. | [02-architecture](02-architecture/index.md) |
| Quality and operations | Verification evidence, configuration, installation, backup, security, and operational unknowns. | [04-quality-operations](04-quality-operations/index.md) |

Planning documents are intentionally omitted; the owner selected **no plan** for this documentation pass.

## Existing user documentation

This baseline complements rather than replaces the end-user documentation:

- [README](../README.md) — product overview and quick start.
- [Wiki home](../wiki/home.md) — user documentation navigation.
- [Installation](../wiki/installation.md), [configuration](../wiki/configuration.md), [CLI](../wiki/cli.md), [TUI](../wiki/tui.md), and [migration](../wiki/migration-from-1x.md).
- [Changelog](../CHANGELOG.md) — release and change history.

## Evidence and authority

- **OBSERVED** means inspected in the current source tree or configuration.
- **DOCUMENTED** means stated in existing project documentation.
- **INFERRED** means an interpretation from those sources, not a separate product decision.
- **TBD** means an owner decision or operational target remains unresolved.

Code describes current implementation; existing product docs describe stated intent; tests and CI are evidence only for behavior they cover. This review inspected revision `d65df2d` and was static: no application deployment or test suite was run. Requirement IDs describe existing capabilities unless explicitly marked otherwise; they do not establish new roadmap commitments. Owner clarifications on 2026-10-04 confirm opt-in public-link control, console permission including power control, optional automatic backups and restore to existing/new instances, and LAN/VPN access. Explicit EULA acceptance before starting a new server with `eula=false` is an accepted target, not implemented behavior. No ADR history or repository `AGENTS.md` was found in the inspected tree. No coding-agent pointer was added because that audience was not requested.


## Maintainer guides

- [Development](04-quality-operations/development.md) — setup, `NAVISERVER_DEV=true`, frontend/backend URLs, debugging and extension points.
- [Technical contracts](02-architecture/contracts.md) — HTTP route inventory, requests/DTOs, WebSocket framing, data model and accepted EULA target.

Follow-up config and frontend API-URL tests passed on 2026-10-04; see [check scope](04-quality-operations/index.md#documentation-pass-checks). The initial baseline remains a static review, and no full runtime validation is claimed. TUI retirement is owner intent, not completed removal.
