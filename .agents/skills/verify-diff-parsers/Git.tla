----------------------------- MODULE Git -----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
Files == {"a", "b"}
VARIABLES phase, selected, ranges, expected, result, steps
vars == <<phase, selected, ranges, expected, result, steps>>
Init == /\ phase = "reading" /\ selected = "none" /\ ranges = <<>>
        /\ expected = [f \in Files |-> {}] /\ result = [f \in Files |-> {}] /\ steps = 0
Select == /\ phase = "reading" /\ steps < 3 /\ \E f \in Files: selected' = f
          /\ steps' = steps + 1 /\ UNCHANGED <<phase, ranges, expected, result>>
Hunk == /\ phase = "reading" /\ steps < 3
        /\ \E start \in 1..2, count \in 0..2:
           /\ IF selected # "none" /\ count > 0
              THEN /\ ranges' = Append(ranges, [file |-> selected, lo |-> start, hi |-> start + count - 1])
                   /\ expected' = [expected EXCEPT ![selected] = @ \cup (start..(start + count - 1))]
              ELSE /\ UNCHANGED <<ranges, expected>>
        /\ steps' = steps + 1 /\ UNCHANGED <<phase, selected, result>>
RangeSet == {ranges[i] : i \in 1..Len(ranges)}
Members(f) == UNION {r.lo..r.hi : r \in {x \in RangeSet : x.file = f}}
Finish == /\ phase = "reading" /\ phase' = "finished"
          /\ result' = [f \in Files |-> Members(f)]
          /\ UNCHANGED <<selected, ranges, expected, steps>>
Terminal == /\ phase = "finished" /\ UNCHANGED vars
Next == Select \/ Hunk \/ Finish \/ Terminal
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in {"reading", "finished"} /\ selected \in Files \cup {"none"} /\ steps \in 0..3
Membership == phase = "finished" => result = expected
NoUnselectedRanges == \A r \in RangeSet: r.file \in Files /\ r.lo <= r.hi
=============================================================================
