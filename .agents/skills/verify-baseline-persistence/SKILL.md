---
name: verify-baseline-persistence
description: Check baseline loading, update eligibility, write failure and reload when changing baseline persistence or engine finalization.
---

# Verify baseline persistence

Synchronized with production at `8df20e1d4be438203689f540acf6138d7d8fb89c`. Inspect `git diff 8df20e1d4be438203689f540acf6138d7d8fb89c -- internal/baseline internal/engine` and uncommitted changes, then run from the repository root:

```sh
.agents/skills/verify-baseline-persistence/check.sh
```

The check requires Go, Java, Python 3, Bash, curl and shasum. It verifies the checksum of TLC 1.7.4, checks `Baseline.tla`, and runs a Go probe against the production baseline and finalization code. Three temporary overlays deliberately break missing-file loading, the mutant-error guard and write-error reporting; each must fail for the expected reason before the unmodified code is checked again. The probe checks that a disabled update does not write, but does not inject a disabled-update fault: both the finalization plan and the update helper independently guard that path.

The model starts with a missing, old or invalid baseline and covers load, eligibility, writing and later reload. Mutation errors and disabled updates do not write; an eligible successful write can be reloaded; a failed write reports failure and may leave unchanged, damaged, or fully updated content. The concrete probe also checks malformed input and a directory read/write failure. Copy `Baseline.tla`, `Baseline.cfg` and `view.mjs` to scratch and run `/statechart` to regenerate the phase chart; same-phase steps are hidden.

This finite abstraction assumes the inputs to finalization and does not model concurrent file access, process crashes, individual filesystem operations, JSON fields or every OS error. It does not claim atomic writes: `os.WriteFile` can truncate or partially write before failing. The reload transition represents a later load, not a load automatically performed by finalization. A passing model is not proof that Go implements it; the probe covers representative real-code cases only. If code changes, update model, probes and synchronization commit together.
