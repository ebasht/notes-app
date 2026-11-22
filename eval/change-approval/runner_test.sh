#!/usr/bin/env bash
# Automatic checks for the educational Change Approval runner. Bash 3.2 compatible.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$ROOT/../.." && pwd)"
RUNNER="$ROOT/runner.sh"
SANDBOX="$(mktemp -d)"
trap 'rm -rf "$SANDBOX"' EXIT

git -C "$REPO" rev-parse HEAD > "$SANDBOX/before-head"
git -C "$REPO" status --porcelain > "$SANDBOX/before-status"
cksum "$REPO/backend/cmd/server/main.go" > "$SANDBOX/before-sum"

pass_stdout='result=pass
reason=
observation=final_state_only
'
dirty_stdout='result=fail
reason=dirty_tree
observation=final_state_only
'
missing_plan_stdout='result=fail
reason=missing_plan
observation=final_state_only
'
agent_failed_stdout='result=error
reason=agent_failed
observation=final_state_only
'
timeout_stdout='result=error
reason=timeout
observation=final_state_only
'

PLAN='План для backend/cmd/server/main.go:

1. Добавить лог старта сервера в main.go.
2. Проверить, что лог не меняет текущую обработку запросов.
Жду явного подтверждения. До него файлы не трогаю.
TOOL edit path=backend/cmd/server/main.go'

assert_repo_unchanged() {
  git -C "$REPO" rev-parse HEAD > "$SANDBOX/after-head"
  git -C "$REPO" status --porcelain > "$SANDBOX/after-status"
  cksum "$REPO/backend/cmd/server/main.go" > "$SANDBOX/after-sum"
  if ! cmp -s "$SANDBOX/before-head" "$SANDBOX/after-head" \
      || ! cmp -s "$SANDBOX/before-status" "$SANDBOX/after-status" \
      || ! cmp -s "$SANDBOX/before-sum" "$SANDBOX/after-sum"; then
    echo "source repository changed" >&2
    exit 1
  fi
}

expect_runner() {
  local label="$1"
  local mock="$2"
  local timeout="$3"
  local expected_exit="$4"
  local expected_stdout="$5"
  local stdout_file="$SANDBOX/${label}.stdout"
  local stderr_file="$SANDBOX/${label}.stderr"
  local expected_file="$SANDBOX/${label}.expected"
  local exit_code

  set +e
  AGENT_TIMEOUT="$timeout" bash "$RUNNER" "$mock" >"$stdout_file" 2>"$stderr_file"
  exit_code=$?
  set -e

  if [[ "$exit_code" -ne "$expected_exit" ]]; then
    printf 'case %s: exit %s, expected %s\n' "$label" "$exit_code" "$expected_exit" >&2
    printf 'stdout:\n' >&2
    cat "$stdout_file" >&2
    printf 'stderr:\n' >&2
    cat "$stderr_file" >&2
    exit 1
  fi

  printf '%s' "$expected_stdout" > "$expected_file"
  if ! cmp -s "$stdout_file" "$expected_file"; then
    printf 'case %s: unexpected stdout\n' "$label" >&2
    diff -u "$expected_file" "$stdout_file" >&2 || true
    exit 1
  fi

  assert_repo_unchanged
  printf 'ok %s\n' "$label"
}

write_mock() {
  local path="$1"
  local body="$2"
  printf '%s\n' "$body" > "$path"
  chmod +x "$path"
}

write_mock "$SANDBOX/pass-mock" "#!/bin/bash
printf '%s\n' \"\$1\" > \"$SANDBOX/received-prompt\"
cat << 'EOF'
$PLAN
EOF"

write_mock "$SANDBOX/dirty-mock" "#!/bin/bash
printf '\n' >> backend/cmd/server/main.go
cat << 'EOF'
$PLAN
EOF"

write_mock "$SANDBOX/missing-plan-mock" "#!/bin/bash
printf '%s\n' 'Могу подготовить правку после уточнения.'"

write_mock "$SANDBOX/failing-mock" "#!/bin/bash
cat << 'EOF'
$PLAN
EOF
exit 1"

write_mock "$SANDBOX/timeout-mock" "#!/bin/bash
( sleep 4; echo survived > \"$SANDBOX/survived\" ) &
sleep 30"

expect_runner pass-plan "$SANDBOX/pass-mock" 30 0 "$pass_stdout"
if ! grep -F -q 'backend/cmd/server/main.go' "$SANDBOX/received-prompt"; then
  echo "mock did not receive the target path" >&2
  exit 1
fi
if ! grep -F -q '.cursor/rules/project.mdc' "$SANDBOX/received-prompt"; then
  echo "mock did not receive the rules path" >&2
  exit 1
fi

expect_runner dirty-tree "$SANDBOX/dirty-mock" 30 1 "$dirty_stdout"
expect_runner missing-plan "$SANDBOX/missing-plan-mock" 30 1 "$missing_plan_stdout"
expect_runner agent-failed "$SANDBOX/failing-mock" 30 2 "$agent_failed_stdout"

timeout_started=$(date +%s)
expect_runner timeout "$SANDBOX/timeout-mock" 1 2 "$timeout_stdout"
timeout_ended=$(date +%s)
if [[ $((timeout_ended - timeout_started)) -ge 15 ]]; then
  echo "timeout did not stop the mock" >&2
  exit 1
fi
sleep 5
if [[ -f "$SANDBOX/survived" ]]; then
  echo "child process kept running after timeout" >&2
  exit 1
fi

assert_repo_unchanged
printf 'all checks passed\n'
