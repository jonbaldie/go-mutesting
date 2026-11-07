#!/usr/bin/env bash
set -Eeuo pipefail
trap 'echo "SETUP ERROR: a required command failed." >&2; exit 2' ERR

skill_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(git -C "$skill_dir" rev-parse --show-toplevel)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/go-mutesting-per-test-identity.XXXXXXXX")
trap 'rm -rf -- "$scratch"' EXIT

python3 - "$repo" "$skill_dir" "$scratch" <<'PY'
import json
import os
import sys

repo, skill, scratch = sys.argv[1:]
overlay = {"Replace": {
    os.path.join(repo, "internal/coverage/zz_identity_probe_test.go"):
    os.path.join(skill, "probe_test.go"),
    os.path.join(repo, "internal/engine/zz_identity_fallback_probe_test.go"):
    os.path.join(skill, "fallback_probe_test.go"),
}}
with open(os.path.join(scratch, "overlay.json"), "w", encoding="utf-8") as f:
    json.dump(overlay, f)
PY

finding=0
incomplete_cli() {
    cat "$1"
    if [ "$finding" -ne 0 ]; then
        echo "FINDING: $2 The earlier property violation remains; the CLI comparison is incomplete." >&2
        exit 1
    fi
    echo "SETUP ERROR: $2" >&2
    exit 2
}

echo 'Checking generated package pairs against the real coverage code...'
if (cd "$repo" && GOMAXPROCS=1 go test -count=1 -overlay "$scratch/overlay.json" \
    -run '^TestGeneratedPackageIdentity$' -v ./internal/coverage) >"$scratch/probe.log" 2>&1; then
    cat "$scratch/probe.log"
    if ! grep -q '^--- PASS: TestGeneratedPackageIdentity ' "$scratch/probe.log" ||
        ! grep -q 'checked 240 ordered pairs;' "$scratch/probe.log"; then
        echo 'SETUP ERROR: the generated-input probe did not run all cases.' >&2
        exit 2
    fi
elif grep -q 'incomplete profiles; first:' "$scratch/probe.log" &&
    grep -q '^--- FAIL: TestGeneratedPackageIdentity ' "$scratch/probe.log" &&
    grep -q 'checked 240 ordered pairs;' "$scratch/probe.log"; then
    cat "$scratch/probe.log"
    finding=1
else
    cat "$scratch/probe.log"
    echo 'SETUP ERROR: the generated-input probe did not complete.' >&2
    exit 2
fi

echo 'Checking that the engine does not filter a missing profile...'
if (cd "$repo" && GOMAXPROCS=1 go test -count=1 -overlay "$scratch/overlay.json" \
    -run '^TestPerTestFallbackLeavesSuiteUnfiltered$' -v ./internal/engine) >"$scratch/fallback.log" 2>&1; then
    cat "$scratch/fallback.log"
    if ! grep -q '^--- PASS: TestPerTestFallbackLeavesSuiteUnfiltered ' "$scratch/fallback.log"; then
        echo 'SETUP ERROR: the fallback probe did not run.' >&2
        exit 2
    fi
elif grep -q 'fallback property violation:' "$scratch/fallback.log" &&
    grep -q '^--- FAIL: TestPerTestFallbackLeavesSuiteUnfiltered ' "$scratch/fallback.log"; then
    cat "$scratch/fallback.log"
    finding=1
else
    cat "$scratch/fallback.log"
    echo 'SETUP ERROR: the fallback probe did not complete.' >&2
    exit 2
fi

echo 'Building the CLI for the end-to-end check...'
if ! (cd "$repo" && GOMAXPROCS=1 go build -o "$scratch/go-mutesting" ./cmd/go-mutesting); then
    echo 'SETUP ERROR: the CLI did not build.' >&2
    exit 2
fi

fixture="$scratch/fixture"
mkdir -p "$fixture/a/b_c" "$fixture/a_b/c"
cat >"$fixture/go.mod" <<'EOF'
module example.test/verify

go 1.22
EOF
cat >"$fixture/mutesting.yml" <<'EOF'
json_output: false
enable_mutators:
  - statement/return
EOF
cat >"$fixture/target.go" <<'EOF'
package verify

func Value() int {
	return 1
}
EOF
cat >"$fixture/target_test.go" <<'EOF'
package verify

import "testing"

func TestRoot(t *testing.T) {}
EOF
cat >"$fixture/a/b_c/sub.go" <<'EOF'
package bc
EOF
cat >"$fixture/a/b_c/sub_test.go" <<'EOF'
package bc

import (
	"testing"
	"example.test/verify"
)

func TestKiller(t *testing.T) {
	if verify.Value() != 1 { t.Fatal("Value changed") }
}
EOF
cat >"$fixture/a_b/c/sub.go" <<'EOF'
package c
EOF
cat >"$fixture/a_b/c/sub_test.go" <<'EOF'
package c

import (
	"testing"
	"example.test/verify"
)

func TestObserver(t *testing.T) { _ = verify.Value() }
EOF

echo 'Checking the same mutant with the full suite and per-test selection...'
if ! (cd "$fixture" && GOMAXPROCS=1 "$scratch/go-mutesting" --workers=1 --exec-timeout=30 \
    --no-diffs --coverage --test-recursive --config=mutesting.yml .) >"$scratch/full.log" 2>&1; then
    incomplete_cli "$scratch/full.log" 'The full-suite control did not complete.'
fi
if ! grep -q '1 killed, 0 escaped, 0 errored' "$scratch/full.log"; then
    incomplete_cli "$scratch/full.log" 'The full-suite control did not kill exactly one mutant.'
fi
if ! (cd "$fixture" && GOMAXPROCS=1 "$scratch/go-mutesting" --workers=1 --exec-timeout=30 \
    --no-diffs --coverage --test-recursive --config=mutesting.yml --per-test .) >"$scratch/per-test.log" 2>&1; then
    incomplete_cli "$scratch/per-test.log" 'The per-test run did not complete.'
fi
if grep -q '1 killed, 0 escaped, 0 errored' "$scratch/per-test.log"; then
    echo 'CLI control: both runs killed the mutant.'
elif grep -q '0 killed, 1 escaped, 0 errored' "$scratch/per-test.log"; then
    cat "$scratch/per-test.log"
    echo 'FINDING: per-test selection let a mutant escape that the full suite killed.' >&2
    finding=1
else
    incomplete_cli "$scratch/per-test.log" 'The per-test run produced an unexpected result.'
fi

if [ "$finding" -ne 0 ]; then
    exit 1
fi
echo 'PASS: generated pairs and the CLI fixture retained the covering tests.'
