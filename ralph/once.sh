#!/bin/bash
set -eo pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <feature>"
  exit 1
fi

feature="$1"

ralph_dir="$(cd "$(dirname "$0")" && pwd)"
workspace_dir="$(dirname "$ralph_dir")"
prompt_file="$ralph_dir/prompt.md"
feature_dir="$workspace_dir/.scratch/$feature"

if [ ! -d "$feature_dir" ]; then
  echo "Feature folder not found: $feature_dir"
  exit 1
fi

select_next_issue() {
  for f in "$feature_dir"/issues/*.md; do
    [ -f "$f" ] || continue
    if grep -q '^\*\*Status:\*\* `ready-for-agent`' "$f"; then
      echo "$f"
      return 0
    fi
  done
  return 1
}

extract_model() {
  local f="$1"
  local m
  m=$(grep -m1 '^\*\*Model:\*\*' "$f" | sed -E 's/^\*\*Model:\*\* `([^`]+)`.*/\1/')
  [ -n "$m" ] && echo "$m" || echo "sonnet"
}

next_issue=$(select_next_issue) || {
  echo "No ready-for-agent issues in $feature_dir/issues/."
  exit 0
}
model=$(extract_model "$next_issue")
echo "ralph: selected $(basename "$next_issue") (model: $model)"

feature_md=$(cat "$feature_dir"/*.md "$feature_dir"/issues/*.md 2>/dev/null || echo "No feature docs found")
commits=$(git log -n 5 --format="%H%n%ad%n%B---" --date=short 2>/dev/null || echo "No commits found")
prompt=$(cat "$prompt_file")

claude --permission-mode acceptEdits --model "$model" \
  "Previous commits: $commits Feature ($feature): $feature_md ASSIGNED ISSUE: $next_issue $prompt"
