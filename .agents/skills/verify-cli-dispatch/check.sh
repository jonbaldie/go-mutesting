#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
skill="$root/.agents/skills/verify-cli-dispatch"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
python3 - "$root" "$skill" "$tmp" <<'PY'
import json, pathlib, sys
root, skill, tmp = map(pathlib.Path, sys.argv[1:])
source = root / 'cmd/go-mutesting/main.go'
code = source.read_text()
needle = 'if opts.General.Version {'
if code.count(needle) != 1: raise SystemExit('Cannot create version precedence mutation; review source')
(tmp / 'broken.go').write_text(code.replace(needle, 'if false && opts.General.Version {'))
probe = {str(root / 'cmd/go-mutesting/verification_overlay_test.go'): str(skill / 'probe_test.go')}
(tmp / 'normal.json').write_text(json.dumps({'Replace': probe}))
probe[str(source)] = str(tmp / 'broken.go')
(tmp / 'broken.json').write_text(json.dumps({'Replace': probe}))
PY
cd "$root"
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/normal.json" -run '^TestVerifyCLI(EarlyExitPrecedence|HandledBeforeErrors)$' ./cmd/go-mutesting
if GOMAXPROCS=1 go test -count=1 -overlay "$tmp/broken.json" -run '^TestVerifyCLI(EarlyExitPrecedence|HandledBeforeErrors)$' ./cmd/go-mutesting >"$tmp/broken.log" 2>&1; then
 echo 'ERROR: broken version precedence passed' >&2; exit 1
fi
if ! grep -q 'precedence bits=' "$tmp/broken.log"; then cat "$tmp/broken.log" >&2; echo 'ERROR: unrelated failure' >&2; exit 1; fi
echo 'PASS: disabled version branch was detected'
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/normal.json" -run '^TestVerifyCLI(EarlyExitPrecedence|HandledBeforeErrors)$' ./cmd/go-mutesting
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then
 curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/tla2tools.jar"
 mv "$tmp/tla2tools.jar" "$jar"
fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Interrupts.tla" "$skill/Interrupts.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Interrupts.tla)
python3 - "$root" "$tmp" <<'PY'
import json, pathlib, sys
root, tmp = map(pathlib.Path, sys.argv[1:])
source = root / 'cmd/go-mutesting/main.go'
code = source.read_text()
needle = 'case <-signals:\n\t\t\tsignal.Stop(signals)'
if code.count(needle) != 1: raise SystemExit('Cannot create signal sensitivity mutation; review source')
(tmp / 'broken-signal.go').write_text(code.replace(needle, 'case <-signals:\n\t\t\t_ = signals'))
(tmp / 'broken-signal.json').write_text(json.dumps({'Replace': {str(source): str(tmp / 'broken-signal.go')}}))
PY
GOMAXPROCS=1 go test -count=1 -run '^TestInterruptLifecycle$' ./cmd/go-mutesting
if GOMAXPROCS=1 go test -count=1 -overlay "$tmp/broken-signal.json" -run '^TestInterruptLifecycle/interrupt$' ./cmd/go-mutesting >"$tmp/broken-signal.log" 2>&1; then
 echo 'ERROR: broken signal interception passed' >&2; exit 1
fi
if ! grep -q 'second signal did not terminate process' "$tmp/broken-signal.log"; then cat "$tmp/broken-signal.log" >&2; echo 'ERROR: unrelated signal failure' >&2; exit 1; fi
echo 'PASS: missing signal stop was detected'
GOMAXPROCS=1 go test -count=1 -run '^TestInterruptLifecycle$' ./cmd/go-mutesting
cp "$skill/Dispatch.tla" "$skill/Dispatch.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config Dispatch.cfg Dispatch.tla)
sed 's/Broken = FALSE/Broken = TRUE/' "$tmp/Dispatch.cfg" > "$tmp/broken-dispatch.cfg"
if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config broken-dispatch.cfg Dispatch.tla) > "$tmp/broken-dispatch-model.log" 2>&1; then echo 'Broken dispatch model passed' >&2; exit 1; fi
grep -q 'Invariant Expected is violated' "$tmp/broken-dispatch-model.log" || { cat "$tmp/broken-dispatch-model.log" >&2; exit 1; }
python3 - "$root" "$tmp" <<'PY'
import json, pathlib, sys
root,tmp=map(pathlib.Path,sys.argv[1:]); source=root/'cmd/go-mutesting/main.go'; s=source.read_text(); old='if len(targets.Files) == 0 {'
if s.count(old)!=1: raise SystemExit('target mutation no longer applies')
(tmp/'target.go').write_text(s.replace(old, 'if false && len(targets.Files) == 0 {'))
overlay=json.loads((tmp/'normal.json').read_text()); overlay['Replace'][str(source)]=str(tmp/'target.go'); (tmp/'target.json').write_text(json.dumps(overlay))
PY
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/normal.json" -run '^TestVerifyCLIDispatch$' ./cmd/go-mutesting
if GOMAXPROCS=1 go test -count=1 -overlay "$tmp/target.json" -run '^TestVerifyCLIDispatch/target_before_baseline$' ./cmd/go-mutesting >"$tmp/target.log" 2>&1; then echo 'Broken target precedence passed' >&2; exit 1; fi
grep -q 'dispatch target before baseline' "$tmp/target.log" || { cat "$tmp/target.log" >&2; exit 1; }
echo 'Detected broken target precedence in model and code'
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/normal.json" -run '^TestVerifyCLIDispatch$' ./cmd/go-mutesting
python3 - "$tmp/Interrupts.tla" <<'PY'
import pathlib,sys
p=pathlib.Path(sys.argv[1]); s=p.read_text(); old='selected = "signal" /\\ ~registered /\\ ~cancelled'; assert s.count(old)==1; p.write_text(s.replace(old,'selected = "signal" /\\ ~cancelled'))
PY
if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config Interrupts.cfg Interrupts.tla) > "$tmp/broken-interrupt-model.log" 2>&1; then echo 'Broken interrupt model passed' >&2; exit 1; fi
grep -q 'Invariant UnregisteredBeforeSignalCancellation is violated' "$tmp/broken-interrupt-model.log" || { cat "$tmp/broken-interrupt-model.log" >&2; exit 1; }
echo 'Detected broken interrupt model'
