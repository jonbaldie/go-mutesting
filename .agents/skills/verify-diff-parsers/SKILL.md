---
name: verify-diff-parsers
description: Check the bounded state-machine behavior of the unified-diff and Git changed-line parsers when either parser changes.
---

# Verify diff parsers

Run this skill after changing the unified-diff parser in `internal/parser` or the changed-line parser in `internal/gitdiff`. An incorrect first changed line or changed-line set can cause mutation testing to include or skip the wrong code.

The checked baseline is `8df20e1d4be438203689f540acf6138d7d8fb89c`. Inspect changes since that commit, including uncommitted changes, and assess whether the generated cases and expectations still describe the implementation's intended behavior:

```sh
git diff 8df20e1d4be438203689f540acf6138d7d8fb89c -- internal/parser internal/gitdiff internal/sourcepath
git status --short
./.agents/skills/verify-diff-parsers/check.sh
```

Run from the repository root. The script requires Go, Java, Python 3, Bash, curl and shasum. It checks the pinned TLC jar, runs both bounded TLA+ models, then runs generated Go probes against the production parsers using temporary overlays. Each probe must detect a deliberately broken production guard, then pass again with the unmodified implementation. The overlays do not change repository source files.

`Unified.tla` starts at a matched header, abstracts invalid ranges and body lines into rejection, bounds declared counts to two, and checks count bounds, retention of the first change against the head of an append-only change history, and acceptance only after complete consumption. It stops reading as soon as both declared counts are consumed; a separate end-of-input action rejects an incomplete body. Markers may occur while the hunk is incomplete. The model generates body events nondeterministically instead of storing a concrete input buffer, fixes the original start to one, and abstracts header search, line splitting and integer overflow. `Git.tla` explores three header or hunk events across two files and checks that collecting positive ranges after selection and flattening them preserves expected membership; a zero-count hunk contributes nothing. TLC visited 143 and 132 distinct states respectively. These models do not execute Go: the probes below provide an independent, finite implementation check. The Git membership oracle and collection use the same modeled eligibility condition; the concrete zero-count fault check helps guard that assumption. Coalescing is represented by membership rather than interval ordering or representation.

For diagrams, copy the models and views to scratch and run `/statechart` on each with its matching config and view. The resulting charts merge states by phase and hide transitions that remain within a phase.

The unified-diff probe enumerates all 364 edit scripts of length zero through five over context, deletion and addition. Its expected first changed original line comes from the edit script, independently of the parser. It also checks count mismatches, LF and CRLF input, and a variant with a no-newline marker after every generated body line. The Git probe enumerates all 9,331 event sequences of length zero through five over two file headers, deletion-only and nonempty hunks, and an irrelevant line. It derives expected changed-line sets independently and checks the public line-membership behavior and file keys.

These are exhaustive checks only within those finite input domains, not an unbounded proof. They do not cover every malformed diff, arbitrary hunk coordinates and counts, integer limits, Git command execution, suffix matching in `internal/sourcepath`, all path syntax, or interactions outside the generated sequences. Run `go test ./internal/sourcepath ./internal/gitdiff` for suffix matching. If the parser contract or reachable states change, extend the input generators and their independent expectations, update the mutation checks when their target guards move, and record the new baseline after reviewing the changes.
