------------------------- MODULE Preparation -------------------------
VARIABLES phase, cancelled, stableUsable, result
CONSTANTS BrokenCancellation, BrokenStable
vars == <<phase, cancelled, stableUsable, result>>
Init == /\ phase = "start" /\ cancelled \in BOOLEAN /\ stableUsable \in BOOLEAN /\ result = "none"
CoverageFailure == /\ phase = "start" /\ phase' = "coverage" /\ result' = (IF cancelled /\ ~BrokenCancellation THEN "cancel" ELSE "fallback") /\ UNCHANGED <<cancelled, stableUsable>>
StableExec == /\ phase = "coverage" /\ result = "fallback" /\ phase' = "done" /\ result' = (IF stableUsable \/ BrokenStable THEN "stable" ELSE "original") /\ UNCHANGED <<cancelled, stableUsable>>
Stop == /\ phase = "coverage" /\ result = "cancel" /\ phase' = "done" /\ UNCHANGED <<cancelled, stableUsable, result>>
Next == CoverageFailure \/ StableExec \/ Stop
Spec == Init /\ [][Next]_vars
CancellationPreserved == phase = "start" \/ ~cancelled \/ result = "cancel"
OriginalFallback == phase # "done" \/ cancelled \/ stableUsable \/ result = "original"
=============================================================================
