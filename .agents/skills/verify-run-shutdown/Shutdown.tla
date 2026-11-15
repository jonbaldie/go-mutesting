--------------------------- MODULE Shutdown ---------------------------
EXTENDS FiniteSets, Naturals
CONSTANT Jobs
VARIABLES pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed
vars == <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Init == /\ pending = Jobs /\ queued = {} /\ active = {} /\ completed = {} /\ skipped = {} /\ dropped = {} /\ cancelled = FALSE /\ producerDone = FALSE /\ closing = FALSE /\ workersDone = FALSE /\ progressDone = FALSE /\ summaries = 0 /\ removed = FALSE
Submit(j) == /\ j \in pending /\ ~producerDone /\ pending' = pending \ {j} /\ queued' = queued \cup {j} /\ UNCHANGED <<active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Drop(j) == /\ j \in pending /\ cancelled /\ ~producerDone /\ pending' = pending \ {j} /\ dropped' = dropped \cup {j} /\ UNCHANGED <<queued, active, completed, skipped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Cancel == /\ ~cancelled /\ cancelled' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, producerDone, closing, workersDone, progressDone, summaries, removed>>
ProducerDone == /\ pending = {} /\ ~producerDone /\ producerDone' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, closing, workersDone, progressDone, summaries, removed>>
Start(j) == /\ j \in queued /\ ~cancelled /\ queued' = queued \ {j} /\ active' = active \cup {j} /\ UNCHANGED <<pending, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Skip(j) == /\ j \in queued /\ cancelled /\ queued' = queued \ {j} /\ skipped' = skipped \cup {j} /\ UNCHANGED <<pending, active, completed, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Complete(j) == /\ j \in active /\ active' = active \ {j} /\ completed' = completed \cup {j} /\ UNCHANGED <<pending, queued, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries, removed>>
Close == /\ producerDone /\ ~closing /\ closing' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, workersDone, progressDone, summaries, removed>>
WaitWorkers == /\ closing /\ queued = {} /\ active = {} /\ ~workersDone /\ workersDone' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, progressDone, summaries, removed>>
StopProgress == /\ workersDone /\ ~progressDone /\ progressDone' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, summaries, removed>>
Summarize == /\ progressDone /\ summaries = 0 /\ summaries' = 1 /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, removed>>
Remove == /\ summaries = 1 /\ ~removed /\ removed' = TRUE /\ UNCHANGED <<pending, queued, active, completed, skipped, dropped, cancelled, producerDone, closing, workersDone, progressDone, summaries>>
Done == /\ removed /\ UNCHANGED vars
Next == (\E j \in Jobs: Submit(j) \/ Drop(j) \/ Start(j) \/ Skip(j) \/ Complete(j)) \/ Cancel \/ ProducerDone \/ Close \/ WaitWorkers \/ StopProgress \/ Summarize \/ Remove \/ Done
Spec == Init /\ [][Next]_vars
SafeSummary == summaries <= 1 /\ (summaries = 1 => producerDone /\ workersDone /\ progressDone /\ active = {} /\ queued = {})
SafeRemoval == removed => summaries = 1
NoLostJobs == Jobs = pending \cup queued \cup active \cup completed \cup skipped \cup dropped
=============================================================================
