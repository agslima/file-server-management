# Documentation Overview

The `docs/` directory contains project-wide documentation for the File Server Management platform. It covers architecture and design references, API and auth guidance, storage behavior, operational practices, security reviews and threat modeling, contributor workflows, and the full set of Architecture Decision Records (ADRs). It also includes a Postman collection for API exploration.

## Validation-first documentation rule

- Baseline capability claims must be recorded in [`capability-ledger.md`](capability-ledger.md).
- Each claim must include a runnable command and expected result.
- README claim IDs should map back to the ledger (`CL-###`).
- Target-state claims should not be presented as baseline-implemented until they have runnable validation.

## Table of Contents

### Index

- [Documentation Overview (this file)](README.md)
- [Capability Ledger (claim -> validation mapping)](capability-ledger.md)
- [Governance (merge gates)](governance.md)
- [Project Alignment Review](project-alignment-review.md)
- [Roadmap (staged milestones)](roadmap.md)

### Getting Started

Baseline onboarding is `./file-engine/scripts/dev.sh`; root `docker-compose.yml` is the canonical compose entry point.

- [Setup & Developer Onboarding](setup.md)

### Architecture & Design

- [Platform Architecture](architecture.md)
- [File Engine Technical Documentation](architecture_file-engine.md)
- [Dataflow Security Risk Assessment](dataflow-security-risk-assessment.md)

### API & Integration

- [API Reference (gRPC + HTTP)](api-reference.md)
- [API Storage + Automatic Authorization Enforcement](api_storage_authz.md)
- [Errors & Status Codes](errors.md)

### Authentication, Authorization & Storage

- [Authentication & Authorization](auth.md)
- [JWT Integration](jwt_integration.md)
- [Storage Backends](storage_backends.md)

### Observability & Operations

- [Observability](observability.md)
- [Production Checklist](prod-checklist.md)
- [Platform Engineers Guide](platform-engineers.md)

### Security & Risk

- [Threat Model](threat-model.md)
- [Security Reviewers Guide](security-reviewers.md)

### Contribution Guides

- [Contributor Guide](contributors.md)

### Tooling

- [Postman Collection](postman_collection.json)

### Architecture Decision Records (ADRs)

- [ADR 0001: ADR Process](adr/0001-adr-process.md)
- [ADR 0002: Hybrid PHP + Go Architecture](adr/0002-hybrid-php-go-architecture.md)
- [ADR 0003: Async Jobs for Mutations](adr/0003-async-jobs-for-mutations.md)
- [ADR 0004: Queue Technology Selection](adr/0004-queue-technology-selection.md)
- [ADR 0005: Service-to-Service Auth](adr/0005-service-to-service-auth.md)
- [ADR 0006: Path Safety Model](adr/0006-path-safety-model.md)
- [ADR 0007: Upload Staging and Malware Gating](adr/0007-upload-staging-and-malware-gating.md)
- [ADR 0008: Observability Requirements](adr/0008-observability-requirements.md)

## Doc ownership & review cadence

Assign owners and review cadence to prevent silent drift. Owners are roles (not individuals) and should update these entries when responsibility changes.

| Document | Owner (role) | Review cadence |
| :-- | :-- | :-- |
| `docs/README.md` | Docs Maintainer | Quarterly |
| `docs/capability-ledger.md` | Project Maintainers | Per release |
| `docs/project-alignment-review.md` | Project Maintainers | Quarterly |
| `docs/governance.md` | Project Maintainers | Quarterly |
| `docs/roadmap.md` | Platform Engineering | Monthly |
| `docs/setup.md` | Developer Experience | Monthly |
| `docs/contributors.md` | Developer Experience | Quarterly |
| `docs/architecture.md` | Platform Engineering | Quarterly |
| `docs/architecture_file-engine.md` | File Engine | Quarterly |
| `docs/api-reference.md` | API/Contract | Per release |
| `docs/api_storage_authz.md` | API/Contract | Per release |
| `docs/errors.md` | API/Contract | Per release |
| `docs/auth.md` | Security/Auth | Quarterly |
| `docs/jwt_integration.md` | Security/Auth | Quarterly |
| `docs/observability.md` | Platform Engineering | Quarterly |
| `docs/platform-engineers.md` | Platform Engineering | Quarterly |
| `docs/storage_backends.md` | Storage/Infra | Quarterly |
| `docs/threat-model.md` | Security | Quarterly |
| `docs/dataflow-security-risk-assessment.md` | Security | Quarterly |
| `docs/security-reviewers.md` | Security | Quarterly |
| `docs/prod-checklist.md` | Security | Per release |
