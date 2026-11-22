#!/usr/bin/env bash
# Educational runner for a mock Change Approval agent. Bash 3.2 compatible.
#
# This stage launches only an executable passed by the caller. It does not
# start Cursor CLI and does not read API keys.
#
# The mock's stdout is stored as reply.txt. events.log is never created, so
# the judge reports observation=final_state_only.
#
# Timeout signals the mock's process group with TERM and then KILL. Children
# that stay in that group are terminated with it. A descendant that starts a
# new session leaves the group. This runner does not walk the process tree
# and does not claim to stop or isolate such descendants.
#
# The temporary copy is not an OS sandbox. The mock is placed in that copy,
# but it still runs as the current user and inherits the parent environment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"
JUDGE="$SCRIPT_DIR/judge.sh"
PROMPT_FILE="$SCRIPT_DIR/prompt.txt"

RUN_DIR=""

cleanup() {
  if [[ -n "$RUN_DIR" && -d "$RUN_DIR" ]]; then
    rm -rf "$RUN_DIR"
  fi
}
trap cleanup EXIT

finish() {
  local code="$1"
  local result="$2"
  local reason="$3"
  printf 'result=%s\nreason=%s\nobservation=final_state_only\n' "$result" "$reason"
  exit "$code"
}

if [[ "$#" -ne 1 ]]; then
  finish 2 error usage
fi

agent="$1"
case "$agent" in
  /*) ;;
  *) finish 2 error bad_agent ;;
esac
if [[ ! -f "$agent" || ! -x "$agent" ]]; then
  finish 2 error bad_agent
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
  finish 2 error bad_prompt
fi

if [[ -z "${AGENT_TIMEOUT:-}" ]]; then
  AGENT_TIMEOUT=180
fi
case "$AGENT_TIMEOUT" in
  [1-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]) ;;
  *) finish 2 error bad_timeout ;;
esac

prompt="$(cat "$PROMPT_FILE")"
if [[ -z "$prompt" ]]; then
  finish 2 error bad_prompt
fi

RUN_DIR="$(mktemp -d)"
WORK="$RUN_DIR/workspace"
RESULT="$RUN_DIR/result"
mkdir -p "$WORK" "$RESULT"

set +e
git -C "$REPO" archive --format=tar HEAD | tar -x -C "$WORK"
archive_status=$?
set -e
if [[ "$archive_status" -ne 0 || ! -f "$WORK/.cursor/rules/project.mdc" ]]; then
  finish 2 error archive_failed
fi

git -C "$WORK" init >/dev/null 2>&1
git -C "$WORK" add -A
git -C "$WORK" -c user.name=eval -c user.email=eval@example.invalid commit -q -m baseline

if [[ -n "$(git -C "$WORK" status --porcelain)" ]]; then
  finish 2 error dirty_baseline
fi

set -m
(
  cd "$WORK" || exit 1
  exec "$agent" "$prompt"
) >"$RESULT/reply.txt" 2>"$RESULT/agent.err" &
agent_pid=$!
set +m

start=$SECONDS
timed_out=0
while kill -0 "$agent_pid" 2>/dev/null; do
  if [[ $((SECONDS - start)) -ge $AGENT_TIMEOUT ]]; then
    timed_out=1
    kill -TERM -"$agent_pid" 2>/dev/null || true
    sleep 1
    kill -KILL -"$agent_pid" 2>/dev/null || true
    break
  fi
  sleep 1
done

set +e
wait "$agent_pid"
agent_status=$?
set -e

if [[ "$timed_out" -eq 1 ]]; then
  finish 2 error timeout
fi
if [[ "$agent_status" -ne 0 ]]; then
  finish 2 error agent_failed
fi

git -C "$WORK" status --porcelain > "$RESULT/status.txt"

set +e
bash "$JUDGE" "$RESULT"
judge_status=$?
set -e
exit "$judge_status"
