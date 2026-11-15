---
name: verify-preparation-fallback
description: Verify coverage-preparation cancellation and stable-executable fallback.
---

# Verify preparation fallback

Run `./.agents/skills/verify-preparation-fallback/check.sh` from the repository root. TLC checks bounded cancellation and fallback choices in `Preparation.tla`, including broken variants. Read its result with `/statechart`. The Go probe checks the real engine's distinction between cancelled and ordinary coverage preparation failure, and executes the real stable wrapper with missing and unusable stable directories. The original executable must receive its arguments and propagate its exit status. Two scratch overlays deliberately suppress cancellation or executable fallback; each must fail for the named reason before the restored probe passes. Go, Java, Python 3 and Bash are required. This check does not change the checkout.

Last matched to `359f0be`. Before running, compare `buildPerTestCoverageProfile`, `perTestUnavailable`, `prepareStableExec`, and `stableExecScript` with this probe, including uncommitted changes. Update the probe for intended changes and only then advance the sync commit. Build failures and unrelated failures do not satisfy sensitivity checks.

The model abstracts each branch as an atomic decision and checks all boolean combinations; the concrete probe ties those decisions to the code. Its sequence does not represent actual lifecycle order: stable-executable preparation happens during initialization, before coverage preparation. Neither is an exhaustive model of discovery, compilation, cancellation timing, executable copying, concurrent workers or the operating system. The per-test batch model and probe live in `/verify-coverage-batch`; normal tests cover cancellation during compilation and profiling, and successful stable file reuse. It does not prove that all failures can be simulated.
