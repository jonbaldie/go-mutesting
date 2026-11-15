---------------------------- MODULE Dispatch ----------------------------
CONSTANT Broken
VARIABLES phase, early, parseBad, completion, configBad, targetFound, baselineBad, info, result
vars == <<phase, early, parseBad, completion, configBad, targetFound, baselineBad, info, result>>
inputs == <<early, parseBad, completion, configBad, targetFound, baselineBad, info>>
Init == /\ phase = "args" /\ early \in {"version", "help", "list", "none"} /\ parseBad \in BOOLEAN /\ completion \in BOOLEAN /\ configBad \in BOOLEAN /\ targetFound \in BOOLEAN /\ baselineBad \in BOOLEAN /\ info \in BOOLEAN /\ result = "none"
Args == /\ phase = "args" /\ UNCHANGED inputs /\ IF early # "none" THEN /\ phase' = "done" /\ result' = early ELSE IF parseBad THEN /\ phase' = "done" /\ result' = "parse-error" ELSE IF completion THEN /\ phase' = "done" /\ result' = "completion" ELSE /\ phase' = "config" /\ result' = result
Config == /\ phase = "config" /\ UNCHANGED inputs /\ IF configBad THEN /\ phase' = "done" /\ result' = "config-error" ELSE /\ phase' = "target" /\ result' = result
Target == /\ phase = "target" /\ UNCHANGED inputs /\ IF ~targetFound /\ ~(Broken /\ baselineBad) THEN /\ phase' = "done" /\ result' = "target-error" ELSE /\ phase' = "baseline" /\ result' = result
Baseline == /\ phase = "baseline" /\ UNCHANGED inputs /\ IF baselineBad THEN /\ phase' = "done" /\ result' = "baseline-error" ELSE /\ phase' = "dispatch" /\ result' = result
Dispatch == /\ phase = "dispatch" /\ UNCHANGED inputs /\ phase' = "done" /\ result' = (IF info THEN "info" ELSE "run")
Next == Args \/ Config \/ Target \/ Baseline \/ Dispatch
Spec == Init /\ [][Next]_vars
Expected == phase # "done" \/ result = (IF early # "none" THEN early ELSE IF parseBad THEN "parse-error" ELSE IF completion THEN "completion" ELSE IF configBad THEN "config-error" ELSE IF ~targetFound THEN "target-error" ELSE IF baselineBad THEN "baseline-error" ELSE IF info THEN "info" ELSE "run")
=============================================================================
