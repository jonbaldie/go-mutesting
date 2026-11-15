#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
skill="$root/.agents/skills/verify-script-outcomes"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/jar"; mv "$tmp/jar" "$jar"; fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Outcomes.tla" "$skill/Outcomes.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config Outcomes.cfg Outcomes.tla)
sed 's/Broken = FALSE/Broken = TRUE/' "$tmp/Outcomes.cfg" > "$tmp/broken.cfg"
if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config broken.cfg Outcomes.tla) >"$tmp/model-broken.log" 2>&1; then echo 'Broken model passed' >&2; exit 1; fi
grep -q 'Invariant .* is violated' "$tmp/model-broken.log" || { cat "$tmp/model-broken.log" >&2; exit 1; }
echo 'Detected broken model'
cp -R "$root/scripts/exec" "$tmp/scripts"
"$skill/probe.sh" "$tmp/scripts"
python3 - "$tmp/scripts/outcome.sh" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
s = p.read_text()
needle = '\t\toutcome=2\n'
if s.count(needle) != 1: raise SystemExit('Cannot construct build-failure sensitivity mutation')
p.write_text(s.replace(needle, '\t\toutcome=1\n'))
PY
if "$skill/probe.sh" "$tmp/scripts" >"$tmp/broken.log" 2>&1; then
  echo 'ERROR: broken build outcome passed' >&2; exit 1
fi
if ! grep -q 'build_failure: expected exit 2, got 0' "$tmp/broken.log"; then
  cat "$tmp/broken.log" >&2; echo 'ERROR: unrelated sensitivity failure' >&2; exit 1
fi
echo 'Detected broken build-failure classification'
"$skill/probe.sh" "$root/scripts/exec"
