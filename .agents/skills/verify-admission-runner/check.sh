#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
skill="$root/.agents/skills/verify-admission-runner"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
jar="${MUTAGO_TLC_JAR:-${XDG_CACHE_HOME:-$HOME/.cache}/mutago-formal-verification/tla2tools-1.7.4.jar}"
mkdir -p "$(dirname "$jar")"
if [ ! -f "$jar" ]; then
  curl -fsSL https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar -o "$tmp/tla2tools.jar"
  mv "$tmp/tla2tools.jar" "$jar"
fi
[ "$(shasum -a 256 "$jar" | cut -d ' ' -f 1)" = 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88 ] || { echo 'TLC jar checksum mismatch' >&2; exit 1; }
cp "$skill/Admission.tla" "$skill/Admission.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Admission.tla)
python3 - "$root" "$tmp" <<'PY'
import json, pathlib, sys
root, tmp = map(pathlib.Path, sys.argv[1:])
model = tmp / 'Admission.tla'
s = model.read_text()
old = 'last\' = "reject" /\\ UNCHANGED <<seen, admitted>>'
if s.count(old) != 1: raise SystemExit('model mutation no longer applies')
model.write_text(s.replace(old, 'last\' = "reject" /\\ seen\' = seen \\cup {k} /\\ UNCHANGED admitted'))
for name, source, old, new in [
 ('admission', root/'internal/engine/scope.go', 'return skipGitDiff', 's.seen[c.checksum] = struct{}{}\n\t\treturn skipGitDiff'),
 ('runner', root/'internal/engine/engine.go', 'case 0:\n\t\treturn 1\n\tcase 1:\n\t\treturn 0', 'case 0:\n\t\treturn 0\n\tcase 1:\n\t\treturn 0'),
]:
 s=source.read_text()
 if s.count(old) != 1: raise SystemExit(name+' mutation no longer applies')
 broken=tmp/(name+'.go'); broken.write_text(s.replace(old,new))
 (tmp/(name+'.json')).write_text(json.dumps({'Replace': {str(source): str(broken)}}))
PY
if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 Admission.tla) >"$tmp/model.log" 2>&1; then
 echo 'broken admission model passed' >&2; exit 1
fi
grep -q 'Invariant OnlyAcceptedAreSeen is violated' "$tmp/model.log" || { cat "$tmp/model.log" >&2; exit 1; }
cd "$root"
GOMAXPROCS=1 go test -count=1 -run '^(TestMutantScopeAdmitDuplicateAndOrder|TestRunnerOutcomes)$' ./internal/engine
for spec in 'admission:TestMutantScopeAdmitDuplicateAndOrder:matching mutant should be admitted' 'runner:TestRunnerOutcomes:builtin_pass_escapes'; do
 IFS=: read -r name test evidence <<< "$spec"
 if GOMAXPROCS=1 go test -count=1 -overlay "$tmp/$name.json" -run "^$test$" ./internal/engine >"$tmp/$name.log" 2>&1; then
   echo "broken $name passed" >&2; exit 1
 fi
 grep -q "$evidence" "$tmp/$name.log" || { cat "$tmp/$name.log" >&2; echo "unrelated failure in $name" >&2; exit 1; }
 echo "Detected broken $name implementation"
done
GOMAXPROCS=1 go test -count=1 -run '^(TestMutantScopeAdmitDuplicateAndOrder|TestRunnerOutcomes)$' ./internal/engine
cp "$skill/Lifecycle.tla" "$skill/Lifecycle.cfg" "$tmp/"
(cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config Lifecycle.cfg Lifecycle.tla)
for fault in BrokenScope BrokenSink BrokenOutcome; do
 sed "s/$fault = FALSE/$fault = TRUE/" "$tmp/Lifecycle.cfg" > "$tmp/lifecycle-broken.cfg"
 if (cd "$tmp" && java -XX:+UseParallelGC -Xmx256m -cp "$jar" tlc2.TLC -workers 1 -config lifecycle-broken.cfg Lifecycle.tla) >"$tmp/lifecycle-$fault.log" 2>&1; then echo "Broken $fault model passed" >&2; exit 1; fi
 case "$fault" in BrokenScope) expected=ScopeOrder;; BrokenSink) expected=SinkOrder;; BrokenOutcome) expected=RunnerResult;; esac
 grep -q "Invariant $expected is violated" "$tmp/lifecycle-$fault.log" || { cat "$tmp/lifecycle-$fault.log" >&2; exit 1; }
 echo "Detected broken model $fault"
done
python3 - "$root" "$skill" "$tmp" <<'PY'
import json,pathlib,sys
root,skill,tmp=map(pathlib.Path,sys.argv[1:]); probe={str(root/'internal/engine/verification_lifecycle_test.go'):str(skill/'lifecycle_probe_test.go')}
(tmp/'lifecycle-normal.json').write_text(json.dumps({'Replace':probe}))
for name,source,old,new in [
 ('scope',root/'internal/engine/scope.go','if gitDiffExcludes(s.gitLines, c.repoFile, c.absFile, c.line) {','if gitDiffExcludes(s.gitLines, c.repoFile, c.absFile, c.line) && c.id == s.runID {'),
 ('sink',root/'internal/engine/mode.go','case opts.General.DryRun:','case opts.General.DryRun && !opts.Exec.NoExec:'),
]:
 s=source.read_text()
 if s.count(old)!=1: raise SystemExit(name+' mutation no longer applies')
 broken=tmp/(name+'-lifecycle.go');broken.write_text(s.replace(old,new)); (tmp/(name+'-lifecycle.json')).write_text(json.dumps({'Replace':{**probe,str(source):str(broken)}}))
PY
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/lifecycle-normal.json" -run '^TestVerify(Scope|Sink)Precedence$' ./internal/engine
for spec in 'scope:Scope:scope precedence' 'sink:Sink:sink precedence'; do
 IFS=: read -r name test expected <<< "$spec"
 if GOMAXPROCS=1 go test -count=1 -overlay "$tmp/$name-lifecycle.json" -run "^TestVerify${test}Precedence$" ./internal/engine >"$tmp/$name-lifecycle.log" 2>&1; then echo "Broken $name passed" >&2; exit 1; fi
 grep -q "$expected" "$tmp/$name-lifecycle.log" || { cat "$tmp/$name-lifecycle.log" >&2; exit 1; }
 echo "Detected broken $name precedence in code"
done
GOMAXPROCS=1 go test -count=1 -overlay "$tmp/lifecycle-normal.json" -run '^TestVerify(Scope|Sink)Precedence$' ./internal/engine
