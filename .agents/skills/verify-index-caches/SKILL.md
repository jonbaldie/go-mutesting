---
name: verify-index-caches
description: Check identifier and safety index clearing and coverage path-cache reuse after changing their lifecycle.
---

# Verify index caches

Run after changing `astutil/index_registry.go`, `astutil/query.go`, `astutil/safety.go`, or `internal/coverage/coverage.go`. Model and probes synced to production commit `7115a33` (the coverage branch also includes the lifecycle tests).

```sh
.agents/skills/verify-index-caches/check.sh
```

The script checks the bounded TLA+ model, verifies four deliberately broken models fail their respective invariants, and reruns the correct model. It then runs each corresponding Go test normally, with a temporary Go overlay that introduces a fault, and normally again. It requires Java, Python 3, Go, curl and shasum. The pinned TLC 1.7.4 jar is downloaded into the user cache if missing and checked by SHA-256. No source file is changed by a probe.

Reproduce the reader-facing statechart in a fresh scratch directory outside the repository:

```sh
/Users/jonathanbaldie/Code/pstack/.agents/skills/statechart/scripts/statechart .agents/skills/verify-index-caches/Indexes.tla --config .agents/skills/verify-index-caches/IndexesChart.cfg --view .agents/skills/verify-index-caches/index-view.mjs --out /tmp/go-mutesting-index-chart
```

The chart uses one node rather than two to stay within the renderer's state budget; both owners and both paths remain. Its labels merge states with the same observable cache status and omit transitions that stay within a label.

## Mapping and limits

`Ensure` abstracts one locked identifier query, including collection of eligible nodes and recording the covered position range. `CoveredNodesCollected` says an abstract range cannot be recorded without collecting its relevant nodes. `ClearIndexes` abstracts `ClearIdentifierCache`, which clears both registries. `BuildSafety` and `ClearSafety` represent building and clearing the safety registry; `SafetyMatchesEpoch` checks that a built index cannot survive a full clear. `Resolve` abstracts each owner's first lookup and cache reuse for a hit or miss; `CachedResolutionCorrect` checks each owner's results against its own immutable abstract source. Owners have different results for the same hit path, so this also checks isolation. `TypeOK` constrains these values.

The bounded two-node, two-path, two-owner model is finite and atomic. It does not model AST eligibility, position boundaries, deduplication, result ordering, suffix selection, mutable input maps, multiple type-checks or races between registry clearing and use of an already-returned pointer. It assumes the underlying type information and coverage source remain stable for the lifetime of their caches. It proves neither concurrency safety nor liveness. The safety epoch is a two-valued abstraction, not a real version counter. `seen` represents relevant collected nodes rather than every traversed AST node.

The Go probes exercise a missing identifier collection, a missing safety clear, bypassed coverage-cache lookup for hits and misses, and a globally shared resolver cache instead of one per profile. The ownership test checks separate `Profile` and `PerTestProfile` instances and stable cached hits; the shared-cache fault is detected by its `Profile` assertion. These tests change their source after caching to expose cache bypass; they do not prove longest-suffix selection or correctness when production coverage data mutate. The probes establish sensitivity to these specific faults, not equivalence between model and implementation.
