--------------------------- MODULE Outcomes ---------------------------
VARIABLES phase, kind, result
CONSTANT Broken
vars == <<phase, kind, result>>
Kinds == {"pass", "test", "build", "setup", "driver", "diagnostic-test", "unknown", "diff-error"}
Init == /\ phase = "start" /\ kind \in Kinds /\ result = "none"
Classify == /\ phase = "start" /\ phase' = "done" /\ UNCHANGED kind
 /\ result' = CASE kind = "pass" -> 1
                 [] kind \in {"test", "diagnostic-test"} -> 0
                 [] kind \in {"build", "setup"} -> IF Broken THEN 0 ELSE 2
                 [] kind \in {"driver", "diff-error"} -> 3
                 [] OTHER -> 7
Next == Classify
Spec == Init /\ [][Next]_vars
Expected == phase = "start" \/ ((kind = "pass" => result = 1)
 /\ (kind \in {"test", "diagnostic-test"} => result = 0)
 /\ (kind \in {"build", "setup"} => result = 2)
 /\ (kind \in {"driver", "diff-error"} => result = 3)
 /\ (kind = "unknown" => result = 7))
=============================================================================
