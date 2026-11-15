--------------------------- MODULE Unified ---------------------------
EXTENDS Naturals, Sequences, TLC
VARIABLES phase, original, revised, oldCount, newCount, first, changes
vars == <<phase, original, revised, oldCount, newCount, first, changes>>
Init == /\ phase = "header" /\ original = 0 /\ revised = 0
        /\ oldCount = 0 /\ newCount = 0 /\ first = 0 /\ changes = <<>>
Header == \/ /\ phase = "header" /\ phase' = "reject"
             /\ UNCHANGED <<original, revised, oldCount, newCount, first, changes>>
          \/ /\ phase = "header" /\ \E o, n \in 0..2:
               /\ oldCount' = o /\ newCount' = n /\ phase' = "body"
             /\ UNCHANGED <<original, revised, first, changes>>
Body == /\ phase = "body"
        /\ ~(original = oldCount /\ revised = newCount)
        /\ \E event \in {"context", "delete", "add", "marker", "invalid"}:
          LET o == IF event \in {"context", "delete"} THEN 1 ELSE 0
              n == IF event \in {"context", "add"} THEN 1 ELSE 0
              change == IF event = "add" /\ oldCount = 0 THEN 2
                        ELSE IF event \in {"delete", "add"} THEN original + 1 ELSE 0
          IN /\ IF event = "invalid" \/ original + o > oldCount \/ revised + n > newCount
                THEN /\ phase' = "reject" /\ UNCHANGED <<original, revised, first, changes>>
                ELSE /\ phase' = "body" /\ original' = original + o /\ revised' = revised + n
                     /\ first' = IF first = 0 /\ change > 0 THEN change ELSE first
                     /\ changes' = IF change > 0 THEN Append(changes, change) ELSE changes
        /\ UNCHANGED <<oldCount, newCount>>
Finish == /\ phase = "body"
          /\ original = oldCount /\ revised = newCount
          /\ phase' = IF first > 0 THEN "success" ELSE "reject"
          /\ UNCHANGED <<original, revised, oldCount, newCount, first, changes>>
EndOfInput == /\ phase = "body"
          /\ ~(original = oldCount /\ revised = newCount)
          /\ phase' = "reject"
          /\ UNCHANGED <<original, revised, oldCount, newCount, first, changes>>
Terminal == /\ phase \in {"success", "reject"} /\ UNCHANGED vars
Next == Header \/ Body \/ Finish \/ EndOfInput \/ Terminal
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in {"header", "body", "success", "reject"}
          /\ original \in 0..2 /\ revised \in 0..2 /\ oldCount \in 0..2 /\ newCount \in 0..2
          /\ first \in 0..3
CountsBounded == original <= oldCount /\ revised <= newCount
FirstStable == first = IF Len(changes) = 0 THEN 0 ELSE Head(changes)
SuccessComplete == phase = "success" => original = oldCount /\ revised = newCount /\ first > 0
=============================================================================
