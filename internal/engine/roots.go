package engine

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jonbaldie/go-mutesting/v2/internal/baseline"
)

// sourceRoots owns the directories that source paths are relative to.
// Mutant identity (IDs, checksums, baselines, agentic JSON) is module-relative.
// Git-diff scope and CI report locations are repository-relative, because git
// and CI systems resolve paths against the repository root, which differs from
// the module root when go.mod lives in a subdirectory.
type sourceRoots struct {
	// module is the module root as `go env GOMOD` reports it. It is kept as
	// reported so mutant IDs stay unchanged.
	module string
	// repo is the git work tree root with symlinks evaluated, or "" outside a
	// git repository.
	repo string
}

// resolveSourceRoots finds both roots once per run.
func resolveSourceRoots(goDir string) sourceRoots {
	return sourceRoots{module: detectModuleRoot(goDir), repo: detectRepoRoot()}
}

// detectRepoRoot returns the root of the git repository containing the working
// directory, the directory that git diff runs in, with symlinks evaluated.
func detectRepoRoot() string {
	out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		return resolved
	}
	return root
}

// ModulePath returns file's identity path, relative to the module root.
func (r sourceRoots) ModulePath(file string) string {
	return baseline.RelPath(file, r.module)
}

// RepoPath returns file relative to the repository root, slash-separated,
// with symlinks evaluated on file. Outside a git repository, or for a file
// outside the repository, it falls back to ModulePath.
func (r sourceRoots) RepoPath(file string) string {
	if r.repo == "" {
		return r.ModulePath(file)
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return r.ModulePath(file)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	rel, err := filepath.Rel(r.repo, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return r.ModulePath(file)
	}
	return filepath.ToSlash(rel)
}
