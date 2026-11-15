#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
skill="$root/.agents/skills/verify-index-caches"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/jar"; mv "$tmp/jar" "$jar"; fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Indexes.tla" "$skill/Indexes.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Indexes.tla)
python3 - "$tmp/Indexes.tla" <<'PYMODEL'
import pathlib, sys
p = pathlib.Path(sys.argv[1]); text = p.read_text()
old = "seen' = seen \cup r"
if text.count(old) != 1: raise SystemExit('cannot create index model mutation')
p.write_text(text.replace(old, "seen' = seen"))
PYMODEL
if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Indexes.tla) >"$tmp/index-model.log" 2>&1; then echo 'broken index model passed' >&2; exit 1; fi
if ! grep -q 'Invariant CoveredNodesCollected is violated' "$tmp/index-model.log"; then cat "$tmp/index-model.log" >&2; exit 1; fi
echo 'Detected broken index model'
for kind in safety path ownership; do
  cp "$skill/Indexes.tla" "$tmp/Indexes.tla"
  python3 - "$tmp/Indexes.tla" "$kind" <<'PYMODEL'
import pathlib, sys
p = pathlib.Path(sys.argv[1]); text = p.read_text()
if sys.argv[2] == 'safety':
 old = "safetyBuilt' = FALSE /\ epoch' = ~epoch"
 new = "UNCHANGED safetyBuilt /\ epoch' = ~epoch"
elif sys.argv[2] == 'path':
 old = "![o][p] = Source(o, p)"
 new = '![o][p] = "missing"'
else:
 old = "![o][p] = Source(o, p)"
 new = "![o][p] = Source(CHOOSE other \\in Owners: other # o, p)"
if text.count(old) != 1: raise SystemExit('cannot create model mutation')
p.write_text(text.replace(old, new))
PYMODEL
  if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Indexes.tla) >"$tmp/$kind-model.log" 2>&1; then echo "broken $kind model passed" >&2; exit 1; fi
  if [[ "$kind" = safety ]]; then expected=SafetyMatchesEpoch; else expected=CachedResolutionCorrect; fi
  if ! grep -q "Invariant $expected is violated" "$tmp/$kind-model.log"; then cat "$tmp/$kind-model.log" >&2; exit 1; fi
  echo "Detected broken $kind model"
done
cp "$skill/Indexes.tla" "$tmp/Indexes.tla"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Indexes.tla)
python3 - "$root" "$tmp" <<'PY'
import json, pathlib, sys
root, tmp = map(pathlib.Path, sys.argv[1:])
for name, source, old, new in [
 ('index', 'astutil/query.go', 'idx.ensure(info, stmts, start, end)', ''),
 ('safety', 'astutil/safety.go', 'indexes.ClearSafety()', 'return'),
 ('path', 'internal/coverage/coverage.go', 'if cached, ok := r.resolved.Load(absFile); ok {', 'if cached, ok := r.resolved.Load(absFile); false && ok {'),
]:
 p=root/source; text=p.read_text()
 if text.count(old)!=1: raise SystemExit('cannot create '+name+' mutation')
 broken=tmp/(name+'.go'); broken.write_text(text.replace(old,new))
 (tmp/(name+'.json')).write_text(json.dumps({'Replace':{str(p):str(broken)}}))
p = root/'internal/coverage/coverage.go'; text = p.read_text()
for old, new in [
 ('type pathResolver[V any] struct {', 'var sharedResolutionCache sync.Map\n\ntype pathResolver[V any] struct {'),
 ('r.resolved.Load(absFile)', 'sharedResolutionCache.Load(absFile)'),
 ('r.resolved.Store(absFile, value)', 'sharedResolutionCache.Store(absFile, value)'),
]:
 if text.count(old) != 1: raise SystemExit('cannot create shared-cache mutation')
 text = text.replace(old, new)
broken = tmp/'ownership.go'; broken.write_text(text)
(tmp/'ownership.json').write_text(json.dumps({'Replace':{str(p):str(broken)}}))
PY
cd "$root"
for item in 'index astutil TestIncrementalIdentifierIndexMatchesFreshAfterClear' 'safety astutil TestSafetyIndexRebuildAfterClear' 'path internal/coverage TestPathResolutionRepeatedHitsAndMisses' 'ownership internal/coverage TestPathResolutionCacheOwnedByProfile' 'hit internal/coverage TestPathResolutionCacheOwnedByProfile'; do
  read -r name package testname <<< "$item"
  GOMAXPROCS=1 GOFLAGS=-buildvcs=false go test -count=1 -run "^${testname}$" "./$package"
  overlay=$name
  if [[ "$name" = hit ]]; then overlay=path; fi
  if GOMAXPROCS=1 GOFLAGS=-buildvcs=false go test -count=1 -overlay "$tmp/$overlay.json" -run "^${testname}$" "./$package" >"$tmp/$name.log" 2>&1; then echo "broken $name passed" >&2; exit 1; fi
  case "$name" in
    index) expected='expected identifier uses';;
    safety) expected='stale safety index';;
    path) expected='cached miss changed';;
    ownership) expected='profile cache leaked across owners';;
    hit) expected='cached profile hit changed';;
  esac
  if ! grep -q "$expected" "$tmp/$name.log"; then cat "$tmp/$name.log" >&2; echo "unrelated $name failure" >&2; exit 1; fi
  echo "Detected broken $name"
  GOMAXPROCS=1 GOFLAGS=-buildvcs=false go test -count=1 -run "^${testname}$" "./$package"
done
