----------------------------- MODULE Indexes -----------------------------
EXTENDS FiniteSets
CONSTANT Nodes, Paths, Owners
VARIABLES seen, covered, safetyBuilt, epoch, safetyEpoch, cached, values
vars == <<seen, covered, safetyBuilt, epoch, safetyEpoch, cached, values>>
Source(o, p) == IF p = "hit" THEN o ELSE "missing"
Init == /\ seen = {} /\ covered = {} /\ safetyBuilt = FALSE /\ epoch = FALSE /\ safetyEpoch = FALSE
        /\ cached = [o \in Owners |-> {}] /\ values = [o \in Owners |-> [p \in Paths |-> "unset"]]
Ensure(r) == /\ r \in SUBSET Nodes
             /\ IF \E old \in covered: r \subseteq old
                   THEN /\ UNCHANGED <<seen, covered>>
                   ELSE /\ seen' = seen \cup r /\ covered' = covered \cup {r}
             /\ UNCHANGED <<safetyBuilt, epoch, safetyEpoch, cached, values>>
BuildSafety == /\ ~safetyBuilt /\ safetyBuilt' = TRUE /\ safetyEpoch' = epoch
               /\ UNCHANGED <<seen, covered, epoch, cached, values>>
ClearSafety == /\ safetyBuilt' = FALSE /\ UNCHANGED <<seen, covered, epoch, safetyEpoch, cached, values>>
ClearIndexes == /\ seen' = {} /\ covered' = {} /\ safetyBuilt' = FALSE /\ epoch' = ~epoch
                /\ UNCHANGED <<safetyEpoch, cached, values>>
Resolve(o, p) == /\ o \in Owners /\ p \in Paths
                 /\ IF p \in cached[o]
                       THEN UNCHANGED <<cached, values>>
                       ELSE /\ cached' = [cached EXCEPT ![o] = @ \cup {p}]
                            /\ values' = [values EXCEPT ![o][p] = Source(o, p)]
                 /\ UNCHANGED <<seen, covered, safetyBuilt, epoch, safetyEpoch>>
Next == (\E r \in SUBSET Nodes: Ensure(r)) \/ BuildSafety \/ ClearSafety \/ ClearIndexes \/ (\E o \in Owners, p \in Paths: Resolve(o, p))
Spec == Init /\ [][Next]_vars
CoveredNodesCollected == \A r \in covered: r \subseteq seen
CachedResolutionCorrect == \A o \in Owners: \A p \in cached[o]: values[o][p] = Source(o, p)
SafetyMatchesEpoch == safetyBuilt => safetyEpoch = epoch
TypeOK == /\ seen \subseteq Nodes /\ covered \subseteq SUBSET Nodes /\ safetyBuilt \in BOOLEAN
          /\ epoch \in BOOLEAN /\ safetyEpoch \in BOOLEAN
          /\ cached \in [Owners -> SUBSET Paths] /\ values \in [Owners -> [Paths -> Owners \cup {"unset", "missing"}]]
=============================================================================
