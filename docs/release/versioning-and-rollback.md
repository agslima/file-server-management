# Release versioning, notes, and rollback

## Versioning strategy

- Use SemVer tags: `vMAJOR.MINOR.PATCH`.
- Merge to `main` stays unreleased until tagged.
- Patch releases: bugfix/hardening only.
- Minor releases: backward-compatible capability promotions.
- Major releases: explicit breaking contract changes with migration notes.

## Release notes discipline

For every release tag:

1. Ensure `CHANGELOG.md` has an entry with:
   - added/changed/fixed/security sections,
   - claim IDs promoted in this release,
   - validation commands run.
2. Tag from `main`:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

## Rollback procedure (documented + drill)

Kubernetes rollback drill script:

```bash
./scripts/drills/k8s_rollback_drill.sh
```

What it proves:

- deployment revision changes are observable,
- rollback action is executable,
- rolled-back template removes the injected probe env mutation.

For a bad rollout, run:

```bash
kubectl -n file-platform rollout undo deployment/file-engine
kubectl -n file-platform rollout status deployment/file-engine --timeout=180s
```
