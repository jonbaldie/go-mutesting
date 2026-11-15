---
name: verify-finalization-indexes
description: Check finalization precedence and remembered generation failures when changing their owners.
---

# Verify finalization

A failed mutation must not be accepted into an updated baseline, and a later successful write must not hide a generation failure.

From the repository root, run `./.agents/skills/verify-finalization-indexes/check.sh`. It requires Go, Java, Python 3 and Bash. It downloads a checksum-pinned TLC jar on first use. TLC explores every combination of five boolean finalization inputs. The real-code checks cover a missing ID, successful baseline update, update with errors, report failure, individual-ID gate bypass and gate failure. They also check that generate mode retains a write failure after a later success. Scratch Go overlays remove the update error guard and forget the write failure. Each must fail its matching behavior test for the named reason; the check then runs the unmodified test again.

The checked production functions are `planFinalization` and `finalizeResults` in `internal/engine/engine.go`, `generateSink.admit` and `finish` in `internal/engine/mode.go`. Inspect changes to these operations, including the working tree, before running. Compare the transitions to `Finalize.tla` and the real-code tests; update the checks for intentional behavior changes. The source baseline is `359f0be`. Advance the sync commit only after checking the code and passing the checks and `/statechart` review.

Read every model result with `/statechart`. Replay a counterexample against the named functions before treating it as a product bug. The TLA+ model collapses baseline update, report writing and quality gates into abstract decisions; it assumes a baseline write succeeds and does not model report or write failures as ordered transitions. The concrete test includes report failure but does not exhaust all output formats, concurrent executions or filesystem failures. A failed build, timeout or unrelated assertion is not a successful sensitivity check.
