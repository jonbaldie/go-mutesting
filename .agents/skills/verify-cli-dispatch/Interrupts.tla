--------------------------- MODULE Interrupts ---------------------------
VARIABLES registered, first, selected, cancelled, cause, closed, terminated
vars == <<registered, first, selected, cancelled, cause, closed, terminated>>
Init == /\ registered = TRUE /\ first = FALSE /\ selected = "none" /\ cancelled = FALSE /\ cause = "none" /\ closed = FALSE /\ terminated = FALSE
First == /\ registered /\ ~first /\ first' = TRUE /\ UNCHANGED <<registered, selected, cancelled, cause, closed, terminated>>
SelectSignal == /\ selected = "none" /\ first /\ selected' = "signal" /\ UNCHANGED <<registered, first, cancelled, cause, closed, terminated>>
SelectDone == /\ selected = "none" /\ cancelled /\ selected' = "done" /\ UNCHANGED <<registered, first, cancelled, cause, closed, terminated>>
Stop == /\ selected # "none" /\ registered /\ registered' = FALSE /\ UNCHANGED <<first, selected, cancelled, cause, closed, terminated>>
CancelSignal == /\ selected = "signal" /\ ~registered /\ ~cancelled /\ cancelled' = TRUE /\ cause' = "signal" /\ UNCHANGED <<registered, first, selected, closed, terminated>>
CloseCancel == /\ ~closed /\ ~cancelled /\ cancelled' = TRUE /\ cause' = "close" /\ UNCHANGED <<registered, first, selected, closed, terminated>>
CloseWait == /\ ~registered /\ cancelled /\ ~closed /\ closed' = TRUE /\ UNCHANGED <<registered, first, selected, cancelled, cause, terminated>>
Second == /\ cancelled /\ ~registered /\ ~terminated /\ terminated' = TRUE /\ UNCHANGED <<registered, first, selected, cancelled, cause, closed>>
Done == /\ closed /\ UNCHANGED vars
Next == First \/ SelectSignal \/ SelectDone \/ Stop \/ CancelSignal \/ CloseCancel \/ CloseWait \/ Second \/ Done
Spec == Init /\ [][Next]_vars
UnregisteredBeforeSignalCancellation == cause = "signal" => ~registered
CloseWaitsForUnregistration == closed => ~registered
=============================================================================
