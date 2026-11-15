---
name: verify-cache-clear
description: Verify that package cache clear invalidates in-flight lazy and bulk loads.
---

# Verify package cache clear

A load started before `ClearPackageCache` may finish for its original caller, but it must not populate either cache after the clear. Otherwise subsequent parses can reuse stale syntax and types.

Run `./.agents/skills/verify-cache-clear/check.sh` from the repository root. It requires Go, Java, Python 3, Bash and a SHA-256-pinned TLC jar (downloaded on first use; `MUTAGO_TLC_JAR` overrides the path). The finite model checks clear and completion interleavings for one outstanding load and two package values. The concrete probe exercises lazy and bulk stale publication and enumerates first-publisher/clear/bulk sequences. Separate scratch overlays remove the lazy and bulk generation checks; each real-loader test must fail for stale publication, and the restored tests must pass. No checkout files are altered by the check.

Last matched to `8df20e1d4be438203689f540acf6138d7d8fb89c`: inspect `packageCache.clear`, `currentVersion`, `packageFor`, `publish`, `prepare`, `PreparePackages`, and `loadPkgForDir` in `internal/parser/parse.go` before every run. Update model and probe for intended behavior and record a new sync commit only after checking them. Read every model result with `/statechart` using `Cache.tla`, `Cache.cfg`, and `view.mjs`. Replay any counterexample against the actual code. A build failure or a failure unrelated to stale publication is not a successful sensitivity check.

The model represents locking and load completion as atomic, not package loading, file contents, multiple simultaneous loads, errors, or liveness. It does not assert that a caller already loading gets refreshed data. The probes exercise actual cache methods and controlled real Go loader interleavings, but do not explore every race schedule.
