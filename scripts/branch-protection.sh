#!/usr/bin/env bash
# Apply branch protection + merge settings to RomanAgaltsev/chaotic.
# Requires: gh auth login (admin on the repo).
set -euo pipefail

REPO="RomanAgaltsev/chaotic"

# Repo merge strategy: squash-only, linear history.
gh api -X PATCH "repos/${REPO}" \
  -F allow_squash_merge=true \
  -F allow_merge_commit=false \
  -F allow_rebase_merge=false \
  -F delete_branch_on_merge=true \
  -F squash_merge_commit_title=PR_TITLE \
  -F squash_merge_commit_message=PR_BODY

# Branch protection on main.
gh api -X PUT "repos/${REPO}/branches/main/protection" \
  --input - <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["test-success", "lint-success", "security-success", "pr-title"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "dismiss_stale_reviews": true
  },
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "restrictions": null
}
JSON

echo "Branch protection applied to ${REPO}."