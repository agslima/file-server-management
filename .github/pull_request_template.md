Summary

<!-- Explain what changed and why. Keep this focused on observable behavior. -->Change classification

- [ ] Bug fix
- [ ] Feature
- [ ] Refactoring
- [ ] Security hardening
- [ ] Infrastructure / CI/CD
- [ ] Documentation
- [ ] Tests

Affected components:

- [ ] Backend (PHP / Laravel)
- [ ] File Engine (Go)
- [ ] API / Protobuf / gRPC
- [ ] Database / migrations
- [ ] Storage / filesystem
- [ ] Workers / queues
- [ ] Docker / infrastructure
- [ ] CI/CD / supply chain
- [ ] Documentation

Risk assessment

Select the highest applicable level.

- [ ] Level 1 — Lightweight: Documentation, comments, formatting, isolated tests, or non-behavioral changes.
- [ ] Level 2 — Standard: Application logic, API endpoints, services, workers, configuration, or database changes.
- [ ] Level 3 — Critical: Authentication/authorization, tenant isolation, filesystem mutations, upload quarantine/scanning, storage policies, or security-sensitive CI/CD changes.

Risk rationale:

<!-- Explain the potential impact and affected trust boundaries. -->Capability Ledger

Related claims:

<!-- Example: CL-002, CL-018, CL-020 -->Ledger impact:

- [ ] No capability claim affected.
- [ ] Existing capability improved or modified.
- [ ] New capability implemented.
- [ ] Target-state capability partially implemented.
- [ ] Existing claim needs correction or clarification.

Evidence and validation:

<!-- Reference the exact ledger entry, validation command, test, or evidence. -->Automated validation

Confirm the applicable CI checks have passed.

- [ ] Build / compilation
- [ ] Unit tests
- [ ] Integration tests
- [ ] Lint / formatting
- [ ] Static analysis
- [ ] Security / dependency scanning
- [ ] Container / infrastructure validation
- [ ] Generated artifact consistency

CI run / evidence:

<!-- Link the GitHub Actions run or identify the relevant checks. -->Locally executed commands:

# List relevant commands and their results.

Not applicable / not executed:

<!-- Explain why an expected check was not applicable or could not be executed. -->Adversarial review

Review the final diff after implementation and automated validation. Scale the depth of review to the risk level.

Core review

- [ ] Correctness: Does the change behave as intended? Are edge cases, failure paths, and concurrency considered?
- [ ] Security: Does it preserve authentication, authorization, tenant isolation, and trust boundaries?
- [ ] Testing: Do tests cover expected behavior, negative cases, and relevant failure scenarios?
- [ ] Maintainability: Are responsibilities, interfaces, error handling, and abstractions appropriate?
- [ ] Contracts and documentation: Are API contracts, configuration, generated artifacts, and relevant documentation consistent?

Conditional specialist review

Check the applicable areas:

- [ ] Filesystem / storage: Path traversal, symlink attacks, TOCTOU, permissions, namespace escape, and tenant isolation.
- [ ] Authentication / authorization: Privilege escalation, identity normalization, ownership checks, and authorization across asynchronous boundaries.
- [ ] Upload lifecycle: Quarantine bypass, scan enforcement, invalid state transitions, duplicate completion, replay, partial uploads, and staged-file promotion.
- [ ] Workers / queues: Idempotency, duplicate execution, retries, recovery, concurrency, task-state consistency, and auditability.
- [ ] API / contracts: Protobuf compatibility, generated artifacts, gRPC/HTTP gateway behavior, Laravel-Go integration, and error semantics.
- [ ] Database / integrity: Transaction boundaries, migration reversibility, referential integrity, and concurrent updates.
- [ ] Performance / resource usage: Streaming behavior, memory consumption, unbounded concurrency, queue latency, and database/storage bottlenecks.
- [ ] Infrastructure / supply chain: Container privileges, exposed services, secrets, dependency integrity, and CI workflow permissions.
- [ ] Observability: Sensitive data exposure, useful correlation fields, and actionable error reporting.

Review findings

- [ ] No actionable findings remain.
- [ ] Findings were fixed and relevant validation was rerun.
- [ ] Deferred or disputed findings are documented below.

Severity| Finding| Resolution / rationale
Critical| | 
High| | 
Medium| | 
Low| | 

Review method:

- [ ] Independent reviewer / sub-agent
- [ ] Adversarial self-review
- [ ] Lightweight review (Level 1)

Review limitations:

<!-- Identify missing independent review, unavailable tooling, or unverified assumptions. -->Post-review verification

- [ ] Relevant build, tests, lint, and static analysis were rerun after fixes.
- [ ] Relevant Capability Ledger validation commands were rerun.
- [ ] Cross-component integration tests were rerun when applicable.
- [ ] Performance validation was performed when the change could materially affect performance.
- [ ] No unintended generated artifacts or unrelated changes remain.

Documentation and operational impact

- [ ] [`docs/capability-ledger.md`](docs/capability-ledger.md) updated when implementation status or evidence changed.
- [ ] [`docs/setup.md`](docs/setup.md) updated when setup or local development procedures changed.
- [ ] Relevant API, architecture, security, or operational documentation updated.
- [ ] No documentation changes required.

Final checklist

- [ ] The diff contains only intended changes.
- [ ] No secrets, credentials, tokens, or sensitive data are included.
- [ ] No unsupported production-readiness or capability claims were introduced.
- [ ] Known limitations and follow-up work are documented.
- [ ] The change complies with the repository root and scoped "AGENTS.md" instructions.

Additional notes

<!-- Include rollout considerations, compatibility notes, known risks, or follow-up work. -->
