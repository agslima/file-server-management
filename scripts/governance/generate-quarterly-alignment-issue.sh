#!/usr/bin/env bash
set -euo pipefail

output_file="${1:-artifacts/quarterly-alignment-checklist.md}"
mkdir -p "$(dirname "$output_file")"

cat >"$output_file" <<'EOF'
# Quarterly Alignment & Sustainability Review

## Required checks
- [ ] Confirm branch-protection required checks are aligned with `docs/branch-protection-mapping.md`.
- [ ] Verify `.github/OWNERS` and `docs/ownership-backup-matrix.md` are current and reviewer rotations are assigned.
- [ ] Run `./scripts/sustainability-metrics.sh` and attach the markdown report artifact to release notes.
- [ ] Run `./scripts/drills/new_maintainer_operability_drill.sh --full` and attach results.
- [ ] Update `README.md`, `docs/project-alignment-review.md`, and `docs/capability-ledger.md` when governance status changes.

## Evidence links
- [ ] CI run URL:
- [ ] Release note URL:
- [ ] Follow-up issues:
EOF

echo "QUARTERLY_ALIGNMENT_CHECKLIST_GENERATED=$output_file"
echo "To open manually: copy this file into a new GitHub issue body."
