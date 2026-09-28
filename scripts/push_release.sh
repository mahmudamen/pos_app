#!/usr/bin/env bash
# Push both POS.Go repos to their GitHub remotes.
#
# Safe to re-run: if there is nothing new to push it says so instead of
# prompting. It refuses to run when tracked files have uncommitted changes,
# because those are the changes you meant to ship and would otherwise be
# silently left behind.
#
# Untracked files are reported but never block. In this workspace
# Flutter/shan0h-pos/ is a deliberately untracked third-party reference clone,
# and a warning that always fires is a warning you learn to ignore.

set -euo pipefail

# Overridable so this works from a clone checked out anywhere.
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
POS_DIR="${POS_DIR:-$REPO_ROOT}"
POS_GO_DIR="${POS_GO_DIR:-$(dirname "$REPO_ROOT")/pos-go}"

log()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
info() { printf '  ----  %s\n' "$*"; }
bad()  { printf '\033[31mERROR:\033[0m %s\n' "$*" >&2; }

# Local branch -> ref on the remote. pos tracks a remote branch literally named
# "origin" (so the tracking ref is origin/origin); pos-go uses the default.
pending() { git -C "$1" rev-list --count "$2"..HEAD 2>/dev/null || echo 0; }

check_clean() {
  local dir="$1" name="$2" dirty
  dirty="$(git -C "$dir" status --porcelain --untracked-files=no)"
  if [[ -n "$dirty" ]]; then
    bad "$name has uncommitted changes to tracked files:"
    printf '%s\n' "$dirty" | sed 's/^/        /' >&2
    return 1
  fi
  return 0
}

log "Checking worktrees for uncommitted work..."
failed=0
for entry in "pos_app|$POS_DIR" "pos-go|$POS_GO_DIR"; do
  IFS='|' read -r name dir <<< "$entry"
  [[ -d "$dir/.git" ]] || { info "$name: $dir is not a git repo, skipping"; continue; }
  check_clean "$dir" "$name" || failed=1
done
if [[ $failed -eq 1 ]]; then
  bad "Commit (or stash) the changes above, then re-run. Nothing was pushed."
  exit 1
fi

work=0
for entry in "pos_app|$POS_DIR|origin/origin" "pos-go|$POS_GO_DIR|origin"; do
  IFS='|' read -r name dir ref <<< "$entry"
  [[ -d "$dir/.git" ]] || continue
  n="$(pending "$dir" "$ref")"
  [[ "$n" -gt 0 ]] && work=1
  if [[ "$n" -gt 0 ]]; then
    info "$name: $n commit(s) to push"
    git -C "$dir" log --oneline "$ref"..HEAD | sed 's/^/        /'
  else
    info "$name: already up to date"
  fi
done

if [[ $work -eq 0 ]]; then
  log "Nothing to push; both remotes are current."
  exit 0
fi

# If a remote is still HTTPS, the push stops at a username/password prompt.
# That prompt is not the place for your account password: GitHub rejected
# password auth for git in 2021, so what it wants is a token, and a token typed
# at a prompt lands in your shell history. Prefer SSH.
for dir in "$POS_DIR" "$POS_GO_DIR"; do
  [[ -d "$dir/.git" ]] || continue
  url="$(git -C "$dir" remote get-url origin 2>/dev/null || true)"
  if [[ "$url" == https://* ]]; then
    info "$(basename "$dir"): origin is HTTPS ($url) -- expect a credential prompt"
  fi
done

log "Pushing pos_app..."
git -C "$POS_DIR" push origin HEAD:origin

log "Pushing pos-go..."
git -C "$POS_GO_DIR" push origin HEAD:origin

log "Both repos pushed. Nothing here deploys anything -- production is a separate step."
