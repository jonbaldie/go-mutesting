package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/loader" //nolint:staticcheck

	"github.com/jonbaldie/go-mutesting/v2/internal/filter"
)

// ParseFile parses the content of the given file and returns the corresponding ast.File node and its file set for positional information.
// If a fatal error is encountered the error return argument is not nil.
func ParseFile(file string) (*ast.File, *token.FileSet, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}

	return ParseSource(data)
}

// ParseSource parses the given source and returns the corresponding ast.File node and its file set for positional information.
// If a fatal error is encountered the error return argument is not nil.
func ParseSource(data interface{}) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()

	src, err := parser.ParseFile(fset, "", data, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, nil, err
	}

	return src, fset, err
}

// ParseAndTypeCheckFile parses and type-checks the given file, and returns everything interesting about the file.
// If a fatal error is encountered the error return argument is not nil.
func ParseAndTypeCheckFile(file string, collectors []filter.NodeCollector) (*ast.File, *token.FileSet, *types.Package, *types.Info, error) {
	fileAbs, err := filepath.Abs(file)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Could not absolute the file path of %q: %v", file, err)
	}
	dir := filepath.Dir(fileAbs)

	// Use `go list` for the import path so it always reflects the module's
	// go.mod declaration, even when the major-version suffix differs from the
	// GOPATH directory structure (e.g. v2 module at a non-/v2 GOPATH path).
	importPath := goListImportPath(dir)

	var conf = loader.Config{
		ParserMode: parser.AllErrors | parser.ParseComments,
	}

	if importPath != "" && importPath != "." {
		conf.Import(importPath)
	} else {
		// testdata packages and edge cases where go list cannot determine a path
		conf.CreateFromFilenames(dir, fileAbs)
	}

	conf.AllowErrors = true
	prog, err := conf.Load()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Could not load package of file %q: %v", file, err)
	}

	pkgInfo := prog.InitialPackages()[0]

	var src *ast.File
	for _, f := range pkgInfo.Files {
		if prog.Fset.Position(f.Pos()).Filename == fileAbs {
			src = f

			break
		}
	}

	if src != nil {
		for _, c := range collectors {
			c.Collect(src, prog.Fset, fileAbs)
		}
	}

	return src, prog.Fset, pkgInfo.Pkg, &pkgInfo.Info, nil
}

// goListImportPath returns the module-aware import path for the package in dir
// by running `go list .`. This is more reliable than go/build.ImportDir when
// the module's major-version suffix differs from the GOPATH directory structure.
func goListImportPath(dir string) string {
	cmd := exec.Command("go", "list", ".")
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
