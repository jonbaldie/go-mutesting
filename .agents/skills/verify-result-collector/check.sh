#!/usr/bin/env bash
set -euo pipefail
skill=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$skill/../../.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/jar"; mv "$tmp/jar" "$jar"; fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Collector.tla" "$skill/Collector.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Collector.tla)
for fault in event report; do
  cp "$skill/Collector.tla" "$tmp/Collector.tla"
  python3 - "$tmp/Collector.tla" "$fault" <<'PYMODEL'
import pathlib, sys
p = pathlib.Path(sys.argv[1]); text = p.read_text()
if sys.argv[2] == 'event':
 old, new = "Counts(recorded')]", "Counts(recorded)]"
else:
 old, new = "reportCounts' = Counts(recorded)", "reportCounts' = [o \\in Outcomes |-> 0]"
if text.count(old) != 1: raise SystemExit('cannot construct model fault')
p.write_text(text.replace(old, new))
PYMODEL
  if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Collector.tla) >"$tmp/model-$fault.log" 2>&1; then
    echo "Undetected $fault model fault" >&2; exit 1
  fi
  if [ "$fault" = event ]; then expected=EventCountsMatch; else expected=SnapshotCountsMatch; fi
  if ! grep -q "Invariant $expected is violated" "$tmp/model-$fault.log"; then cat "$tmp/model-$fault.log" >&2; exit 1; fi
  echo "Detected $fault model fault"
done
python3 - "$root" "$skill" "$tmp" <<'PY'
import json,pathlib,sys
root,skill,tmp=map(pathlib.Path,sys.argv[1:])
src=root/'internal/engine/results.go'
original=src.read_text()
probe={str(root/'internal/engine/verification_collector_test.go'):str(skill/'probe_test.go')}
(tmp/'normal.json').write_text(json.dumps({'Replace':probe}))
for name,old,new in [
 ('bucket', 'return &c.escaped', 'return &c.killed'),
 ('event', 'Stats: stats}', 'Stats: models.Stats{KilledCount: stats.KilledCount}}'),
 ('count', 'report.Stats.EscapedCount = stats.EscapedCount', 'report.Stats.EscapedCount = 0'),
 ('alias', 'report.Escaped = cloneMutants(c.escaped)', 'report.Escaped = c.escaped'),
]:
 if original.count(old)!=1: raise SystemExit('cannot create '+name+' fault')
 broken=tmp/(name+'.go');broken.write_text(original.replace(old,new))
 (tmp/(name+'.json')).write_text(json.dumps({'Replace':{**probe,str(src):str(broken)}}))
PY
cd "$root"
run() { GOMAXPROCS=1 GOFLAGS=-buildvcs=false go test -count=1 -overlay "$tmp/$1.json" -run '^TestVerifyCollector$' ./internal/engine; }
run normal
for spec in 'bucket:event stats' 'event:event stats' 'count:report count' 'alias:snapshot alias'; do
 name=${spec%%:*}; expected=${spec#*:}
 if run "$name" >"$tmp/$name.log" 2>&1; then echo "Failed to detect $name fault" >&2; exit 1; fi
 if ! grep -q "$expected" "$tmp/$name.log"; then cat "$tmp/$name.log" >&2; echo "unrelated $name failure" >&2; exit 1; fi
 echo "Detected $name fault"
done
run normal
