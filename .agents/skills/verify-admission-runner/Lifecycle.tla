--------------------------- MODULE Lifecycle ---------------------------
CONSTANTS BrokenScope, BrokenSink, BrokenOutcome
VARIABLES phase, git, id, blocked, duplicate, dry, generate, custom, testClass, reason, sink, outcome
vars == <<phase, git, id, blocked, duplicate, dry, generate, custom, testClass, reason, sink, outcome>>
inputs == <<git, id, blocked, duplicate, dry, generate, custom, testClass>>
Init == /\ phase = "scope" /\ git \in BOOLEAN /\ id \in BOOLEAN /\ blocked \in BOOLEAN /\ duplicate \in BOOLEAN /\ dry \in BOOLEAN /\ generate \in BOOLEAN /\ custom \in BOOLEAN /\ testClass \in {"pass", "failure", "build", "error"} /\ reason = "none" /\ sink = "none" /\ outcome = "none"
Scope == /\ phase = "scope" /\ UNCHANGED <<inputs, sink, outcome>> /\ reason' = (IF git /\ ~BrokenScope THEN "git" ELSE IF id THEN "id" ELSE IF blocked THEN "blacklist" ELSE IF duplicate THEN "duplicate" ELSE IF git THEN "git" ELSE "admit") /\ phase' = "sink"
Sink == /\ phase = "sink" /\ UNCHANGED <<inputs, reason, outcome>> /\ sink' = (IF reason # "admit" THEN "none" ELSE IF dry /\ ~BrokenSink THEN "dry" ELSE IF generate THEN "generate" ELSE IF dry THEN "dry" ELSE "execute") /\ phase' = "runner"
Runner == /\ phase = "runner" /\ UNCHANGED <<inputs, reason, sink>> /\ outcome' = (IF sink # "execute" THEN "none" ELSE IF testClass = "error" THEN "errored" ELSE IF testClass = "build" THEN "skipped" ELSE IF (testClass = "pass") = custom \/ BrokenOutcome THEN "killed" ELSE "escaped") /\ phase' = "done"
Next == Scope \/ Sink \/ Runner
Spec == Init /\ [][Next]_vars
ScopeOrder == phase = "scope" \/ reason = (IF git THEN "git" ELSE IF id THEN "id" ELSE IF blocked THEN "blacklist" ELSE IF duplicate THEN "duplicate" ELSE "admit")
SinkOrder == phase \in {"scope", "sink"} \/ sink = (IF reason # "admit" THEN "none" ELSE IF dry THEN "dry" ELSE IF generate THEN "generate" ELSE "execute")
RunnerResult == phase # "done" \/ outcome = (IF sink # "execute" THEN "none" ELSE IF testClass = "error" THEN "errored" ELSE IF testClass = "build" THEN "skipped" ELSE IF (testClass = "pass") = custom THEN "killed" ELSE "escaped")
=============================================================================
