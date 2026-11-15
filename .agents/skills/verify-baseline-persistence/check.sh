#!/usr/bin/env bash
set -euo pipefail
skill=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$skill/../../.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then
 curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/tla2tools.jar"
 mv "$tmp/tla2tools.jar" "$jar"
fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Baseline.tla" "$skill/Baseline.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Baseline.tla)
python3 - "$root" "$skill" "$tmp" <<'PY'
import json, pathlib, sys
root, skill, tmp = map(pathlib.Path, sys.argv[1:])
probe = {str(root/'internal/engine/verification_persistence_test.go'): str(skill/'probe_test.go')}
(tmp/'normal.json').write_text(json.dumps({'Replace': probe}))
for name, path, old, new in [
 ('missing', 'internal/baseline/baseline.go', 'if errors.Is(err, os.ErrNotExist) {', 'if errors.Is(err, os.ErrNotExist) && false {'),
 ('errors', 'internal/engine/engine.go', 'if opts.Baseline.Update && report.Stats.ErrorCount == 0 {', 'if opts.Baseline.Update && report.Stats.ErrorCount >= 0 {'),
 ('write', 'internal/engine/engine.go', 'if err := baseline.Write(opts.Baseline.File, report.Escaped, moduleRoot); err != nil {', 'if err := baseline.Write(opts.Baseline.File, report.Escaped, moduleRoot); false && err != nil {'),
]:
 source = root/path
 text = source.read_text()
 if text.count(old) != 1: raise SystemExit('Cannot construct ' + name + ' mutation')
 broken = tmp/(name+'.go')
 broken.write_text(text.replace(old, new))
 (tmp/(name+'.json')).write_text(json.dumps({'Replace': {**probe, str(source): str(broken)}}))
PY
cd "$root"
run() { GOFLAGS=-buildvcs=false GOMAXPROCS=1 go test -count=1 -overlay "$tmp/$1.json" -run '^TestVerificationBaselinePersistence$' ./internal/engine; }
run normal
for spec in 'missing:missing load' 'errors:errors exit' 'write:write failure exit'; do
 name=${spec%%:*}; expected=${spec#*:}
 if run "$name" >"$tmp/$name.log" 2>&1; then echo "Failed to detect $name fault" >&2; exit 1; fi
 if ! grep -q "$expected" "$tmp/$name.log"; then cat "$tmp/$name.log" >&2; exit 1; fi
 echo "Detected $name fault"
done
run normal
