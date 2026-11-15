---------------------------- MODULE Collector ----------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Mutants, Outcomes
VARIABLES kind, recorded, captured, eventCounts, sent, report, reportCounts, reportReady, summarySent
vars == <<kind, recorded, captured, eventCounts, sent, report, reportCounts, reportReady, summarySent>>
Init == /\ kind \in [Mutants -> Outcomes]
        /\ recorded = [o \in Outcomes |-> {}]
        /\ captured = [m \in Mutants |-> {}]
        /\ eventCounts = [m \in Mutants |-> [o \in Outcomes |-> 0]]
        /\ sent = {}
        /\ report = [o \in Outcomes |-> {}]
        /\ reportCounts = [o \in Outcomes |-> 0]
        /\ reportReady = FALSE
        /\ summarySent = FALSE
AllRecorded == UNION {recorded[o] : o \in Outcomes}
Counts(s) == [o \in Outcomes |-> Cardinality(s[o])]
Add(m) == /\ m \notin AllRecorded
          /\ recorded' = [recorded EXCEPT ![kind[m]] = @ \cup {m}]
          /\ captured' = [captured EXCEPT ![m] = AllRecorded \cup {m}]
          /\ eventCounts' = [eventCounts EXCEPT ![m] = Counts(recorded')]
          /\ UNCHANGED <<kind, sent, report, reportCounts, reportReady, summarySent>>
Send(m) == /\ m \in AllRecorded \ sent
           /\ sent' = sent \cup {m}
           /\ UNCHANGED <<kind, recorded, captured, eventCounts, report, reportCounts, reportReady, summarySent>>
Fill == /\ ~reportReady
             /\ report' = recorded
             /\ reportCounts' = Counts(recorded)
             /\ reportReady' = TRUE
             /\ UNCHANGED <<kind, recorded, captured, eventCounts, sent, summarySent>>
SendSummary == /\ reportReady /\ ~summarySent
             /\ summarySent' = TRUE
             /\ UNCHANGED <<kind, recorded, captured, eventCounts, sent, report, reportCounts, reportReady>>
Next == (\E m \in Mutants: Add(m) \/ Send(m)) \/ Fill \/ SendSummary
Spec == Init /\ [][Next]_vars
CorrectBucket == \A o \in Outcomes: \A m \in recorded[o]: kind[m] = o
EventsIncludeOwnResult == \A m \in sent: m \in captured[m]
EventSnapshotWasRecorded == \A m \in sent: captured[m] \subseteq AllRecorded
EventCountsMatch == \A m \in sent: \A o \in Outcomes:
  eventCounts[m][o] = Cardinality(captured[m] \cap {x \in Mutants: kind[x] = o})
SnapshotCountsMatch == reportCounts = Counts(report)
=============================================================================
