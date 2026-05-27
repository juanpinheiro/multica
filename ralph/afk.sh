#!/bin/bash
set -eo pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <feature> [iterations]"
  exit 1
fi

feature="$1"
iterations="${2:-0}"  # 0 means run until NO MORE TASKS

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

# jq filter: stream the init model line, assistant text, and a short summary
# of every tool call so long tool-only phases don't look like a hang.
stream_text='if (.type == "system" and .subtype == "init") then "ralph: claude running model=" + (.model // "unknown") elif .type == "assistant" then (.message.content[]? | (if .type == "text" then .text elif .type == "tool_use" then "› " + .name + (if .input.file_path then ": " + (.input.file_path | tostring) elif .input.command then ": " + (.input.command | tostring | gsub("\n"; " ¶ ") | .[0:120]) elif .input.pattern then ": " + (.input.pattern | tostring) elif .input.url then ": " + (.input.url | tostring) else "" end) else empty end)) else empty end | gsub("\n"; "\r\n") | . + "\r\n\n"'

# jq filter to extract final result
final_result='select(.type == "result").result // empty'

i=0
while (( iterations == 0 || i < iterations )); do
  i=$((i+1))

  next_issue=$(select_next_issue) || {
    echo "Ralph complete after $((i-1)) iterations. No ready-for-agent issues left."
    exit 0
  }
  model=$(extract_model "$next_issue")
  echo "ralph[$i]: selected $(basename "$next_issue") (model: $model)"

  tmpfile=$(mktemp)
  trap "rm -f $tmpfile" EXIT

  commits=$(git log -n 5 --format="%H%n%ad%n%B---" --date=short 2>/dev/null || echo "No commits found")
  feature_md=""
  for f in "$feature_dir"/*.md "$feature_dir"/issues"/"*.md; do
    [ -f "$f" ] || continue
    feature_md+="=== FILE: $f ==="$'\n'
    feature_md+="$(cat "$f")"$'\n\n'
  done
  [ -z "$feature_md" ] && feature_md="No feature docs found"
  prompt=$(cat "$prompt_file")

  printf '%s\n\n%s\n\n%s\n\n=== ASSIGNED ISSUE: %s ===\n\n%s\n' \
    "Previous commits: $commits" \
    "Feature ($feature):" \
    "$feature_md" \
    "$next_issue" \
    "$prompt" \
  | claude \
      --permission-mode bypassPermissions \
      --model "$model" \
      --verbose \
      --print \
      --output-format stream-json \
  | grep --line-buffered '^{' \
  | tee "$tmpfile" \
  | jq --unbuffered -rj "$stream_text"

  result=$(jq -r "$final_result" "$tmpfile")

  if [[ "$result" == *"<promise>NO MORE TASKS</promise>"* ]]; then
    echo "Ralph complete after $i iterations."
    exit 0
  fi
done
