#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
skill="$root/.agents/skills/verify-preparation-fallback"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/jar"; mv "$tmp/jar" "$jar"; fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Preparation.tla" "$skill/Preparation.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config Preparation.cfg Preparation.tla)
for fault in BrokenCancellation BrokenStable; do
 sed "s/$fault = FALSE/$fault = TRUE/" "$tmp/Preparation.cfg" > "$tmp/broken.cfg"
 if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config broken.cfg Preparation.tla) >"$tmp/model-broken.log" 2>&1; then echo "Broken model $fault passed" >&2; exit 1; fi
 grep -q 'Invariant .* is violated' "$tmp/model-broken.log" || { cat "$tmp/model-broken.log" >&2; exit 1; }
 echo "Detected broken model $fault"
done
python3 - "$root" "$skill" "$tmp" <<'PY'
import json, pathlib, sys
root, skill, tmp = map(pathlib.Path, sys.argv[1:])
p = root / 'internal/engine/engine.go'
s = p.read_text()
changes = {
 'cancel': ('if ctxErr := ctx.Err(); ctxErr != nil {\n\t\treturn nil, ctxErr\n\t}\n\tconsole.Verbose(opts, "Per-test coverage unavailable', 'if false {\n\t\treturn nil, ctx.Err()\n\t}\n\tconsole.Verbose(opts, "Per-test coverage unavailable'),
 'stable': ('exec "$src" "$@"\n`', 'exit 77\n`'),
}
base = {str(root / 'internal/engine/verification_fallback_test.go'): str(skill / 'probe_test.go')}
(tmp / 'normal.json').write_text(json.dumps({'Replace': base}))
for name, (old, new) in changes.items():
 if s.count(old) != 1: raise SystemExit('Cannot construct ' + name + ' mutation')
 broken = tmp / (name + '.go'); broken.write_text(s.replace(old, new))
 (tmp / (name + '.json')).write_text(json.dumps({'Replace': {**base, str(p): str(broken)}}))
PY
cd "$root"
GOMAXPROCS=1 go test -count=1 -timeout 60s -overlay "$tmp/normal.json" -run '^TestVerify(CoverageFallback|StableExecFallback)$' ./internal/engine
for variant in cancel stable; do
 if GOMAXPROCS=1 go test -count=1 -timeout 60s -overlay "$tmp/$variant.json" -run '^TestVerify(CoverageFallback|StableExecFallback)$' ./internal/engine >"$tmp/$variant.log" 2>&1; then
  echo "ERROR: broken $variant passed" >&2; exit 1
 fi
 case "$variant" in cancel) pattern='cancellation lost';; stable) pattern='fallback failed';; esac
 grep -q "$pattern" "$tmp/$variant.log" || { cat "$tmp/$variant.log" >&2; echo 'ERROR: unrelated failure' >&2; exit 1; }
 echo "Detected broken $variant fallback"
done
GOMAXPROCS=1 go test -count=1 -timeout 60s -overlay "$tmp/normal.json" -run '^TestVerify(CoverageFallback|StableExecFallback)$' ./internal/engine
