package coverage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modulePath is the module root — shorter than the package path so that stripping
// it from coverage entries leaves multi-component relative keys.
const modulePath = "github.com/example"

// profile with covered lines 10-15 in foo.go, uncovered 20-25, and bar.go 5-8.
// All entries live inside the "pkg" sub-package of the module.
const sampleProfile = `mode: set
github.com/example/pkg/foo.go:10.5,15.3 3 1
github.com/example/pkg/foo.go:20.1,25.5 2 0
github.com/example/pkg/bar.go:5.1,8.3 1 2
`

func writeTmpProfile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "cover*.out")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestParseProfile_CoveredLines(t *testing.T) {
	path := writeTmpProfile(t, sampleProfile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)

	absFile := "/home/user/src/github.com/example/pkg/foo.go"

	for l := 10; l <= 15; l++ {
		assert.True(t, p.IsCovered(absFile, l), "line %d should be covered", l)
	}
	// hitCount=0 → uncovered
	for l := 20; l <= 25; l++ {
		assert.False(t, p.IsCovered(absFile, l), "line %d should not be covered", l)
	}
}

func TestParseProfile_UncoveredLines(t *testing.T) {
	path := writeTmpProfile(t, sampleProfile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)

	absFile := "/home/user/src/github.com/example/pkg/foo.go"
	assert.False(t, p.IsCovered(absFile, 1))
	assert.False(t, p.IsCovered(absFile, 100))
}

func TestParseProfile_SecondFile(t *testing.T) {
	path := writeTmpProfile(t, sampleProfile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)

	absFile := "/home/user/src/github.com/example/pkg/bar.go"
	assert.True(t, p.IsCovered(absFile, 5))
	assert.True(t, p.IsCovered(absFile, 7))
	assert.False(t, p.IsCovered(absFile, 9))
}

func TestParseProfile_MissingFile(t *testing.T) {
	_, err := ParseProfile("/nonexistent/file.out", modulePath)
	assert.Error(t, err)
}

func TestParseProfile_EmptyProfile(t *testing.T) {
	path := writeTmpProfile(t, "mode: set\n")
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)
	assert.False(t, p.IsCovered("/any/file.go", 1))
}

func TestParseProfile_ModeOnlyLine(t *testing.T) {
	path := writeTmpProfile(t, "mode: atomic\n")
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestIsCovered_UnknownFile(t *testing.T) {
	path := writeTmpProfile(t, sampleProfile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)
	assert.False(t, p.IsCovered("/some/unknown/file.go", 10))
}

func TestIsCovered_DifferentPackageSameFilename(t *testing.T) {
	// A file in a different package with the same name must NOT match.
	path := writeTmpProfile(t, sampleProfile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)

	// The profile has "foo.go" relative key; this abs path resolves to a
	// different module's foo.go.
	absFile := "/home/user/src/github.com/other/module/foo.go"
	assert.False(t, p.IsCovered(absFile, 10))
}

func TestParseProfile_MultipleHits(t *testing.T) {
	profile := "mode: count\ngithub.com/example/pkg/a.go:1.1,3.5 1 5\n"
	path := writeTmpProfile(t, profile)
	p, err := ParseProfile(path, modulePath)
	require.NoError(t, err)

	absFile := "/src/github.com/example/pkg/a.go"
	assert.True(t, p.IsCovered(absFile, 1))
	assert.True(t, p.IsCovered(absFile, 2))
	assert.True(t, p.IsCovered(absFile, 3))
	assert.False(t, p.IsCovered(absFile, 4))
}

func TestParseProfile_NoModulePrefix(t *testing.T) {
	// Coverage entry without the module prefix should be stored as-is.
	profile := "mode: set\nfoo.go:1.1,5.3 2 1\n"
	path := writeTmpProfile(t, profile)
	p, err := ParseProfile(path, "github.com/some/module")
	require.NoError(t, err)
	assert.True(t, p.IsCovered("/abs/path/to/foo.go", 1))
}
