#!/usr/bin/env bash
set -euo pipefail

# Local and PR checks use changed lines. Master CI passes --full.
base="${1:-origin/master}"
set --
if [[ "$base" != --full ]]; then
	git rev-parse --verify "$base^{commit}" >/dev/null
	set -- --git-diff-lines --git-diff-base "$base" --ignore-msi-with-no-mutations
fi

temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT
export GOCACHE="${GOCACHE:-$temporary_directory/cache}"
binary="${GO_MUTESTING_BIN:-$temporary_directory/go-mutesting}"
if [[ -z "${GO_MUTESTING_BIN:-}" ]]; then
	go build -o "$binary" ./cmd/go-mutesting
fi
# Keep existing reports intact.
printf 'json_output: false\n' > "$temporary_directory/config.yml"

# Targets have isolated unit tests. CLI tests recurse; importing and parser
# depend on package-loading test infrastructure.
"$binary" \
	--config "$temporary_directory/config.yml" \
	--workers "${GO_MUTESTING_WORKERS:-1}" \
	--exec-timeout 30 \
	--coverage \
	"$@" \
	--min-msi 75 \
	--min-covered-msi 80 \
	github.com/jonbaldie/go-mutesting/v2/mutator/arithmetic \
	github.com/jonbaldie/go-mutesting/v2/mutator/branch \
	github.com/jonbaldie/go-mutesting/v2/mutator/composite \
	github.com/jonbaldie/go-mutesting/v2/mutator/concurrency \
	github.com/jonbaldie/go-mutesting/v2/mutator/conditional \
	github.com/jonbaldie/go-mutesting/v2/mutator/expression \
	github.com/jonbaldie/go-mutesting/v2/mutator/loop \
	github.com/jonbaldie/go-mutesting/v2/mutator/numbers \
	github.com/jonbaldie/go-mutesting/v2/mutator/select \
	github.com/jonbaldie/go-mutesting/v2/mutator/statement \
	github.com/jonbaldie/go-mutesting/v2/internal/filter \
	github.com/jonbaldie/go-mutesting/v2/internal/coverage \
	github.com/jonbaldie/go-mutesting/v2/internal/gitdiff \
	github.com/jonbaldie/go-mutesting/v2/internal/models
