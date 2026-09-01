#!/usr/bin/env bash
set -Eeuo pipefail

repo_root=${1:?repository root is required}
runner=${2:?runner path is required}
output=${3:?caller-owned output is required}
runner_digest=${RUNNER_DIGEST:?runner digest is required}
subject_sha=${SUBJECT_SHA:-unknown}
toolchain_version=${TOOLCHAIN_VERSION:-go1.27.0}

test "$(git -C "$repo_root" status --porcelain)" = ""
"$runner" synthesize \
  --source "$repo_root/.gooo/metamorphic-counterexample-synthesizer.gooo" \
  --contract "$repo_root/contracts/metamorphic-counterexample-synthesizer-v1.json" \
  --fixtures "$repo_root/fixtures/scenarios.json" \
  --repo-root "$repo_root" \
  --out "$output" \
  --subject-sha "$subject_sha" \
  --toolchain-version "$toolchain_version" \
  --runner-digest "$runner_digest"
test "$(git -C "$repo_root" status --porcelain)" = ""
