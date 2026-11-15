-------------------------- MODULE Baseline --------------------------
EXTENDS TLC
VARIABLES phase, disk, initial, captured, update, errors, writeOK, wrote
vars == <<phase, disk, initial, captured, update, errors, writeOK, wrote>>
Init == /\ phase = "load" /\ disk \in {"missing", "old", "bad"} /\ initial = disk
        /\ captured = "none" /\ update \in BOOLEAN /\ errors \in BOOLEAN
        /\ writeOK \in BOOLEAN /\ wrote = FALSE
Load == /\ phase = "load"
        /\ phase' = IF disk = "bad" THEN "failed" ELSE "run"
        /\ captured' = IF disk = "old" THEN "old" ELSE "none"
        /\ UNCHANGED <<disk, initial, update, errors, writeOK, wrote>>
Run == /\ phase = "run"
       /\ IF ~update \/ errors
          THEN /\ phase' = "done" /\ UNCHANGED <<disk, wrote>>
          ELSE /\ wrote' = TRUE
               /\ IF writeOK THEN /\ phase' = "reload" /\ disk' = "new"
                  ELSE /\ phase' = "failed" /\ disk' \in {disk, "bad", "new"}
       /\ UNCHANGED <<initial, captured, update, errors, writeOK>>
Reload == /\ phase = "reload" /\ captured' = disk /\ phase' = "done"
          /\ UNCHANGED <<disk, initial, update, errors, writeOK, wrote>>
Terminal == /\ phase \in {"done", "failed"} /\ UNCHANGED vars
Next == Load \/ Run \/ Reload \/ Terminal
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in {"load", "run", "reload", "done", "failed"}
          /\ disk \in {"missing", "old", "bad", "new"} /\ captured \in {"none", "old", "new"}
NoIneligibleWrite == (~update \/ errors) => ~wrote /\ disk = initial
ReloadCorrect == (phase = "done" /\ wrote) => captured = "new" /\ disk = "new"
=============================================================================
