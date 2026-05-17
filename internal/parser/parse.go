package parser

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"

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

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
		Dir:  dir,
		Fset: fset,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments)
		},
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Could not load package of file %q: %v", file, err)
	}

	if len(pkgs) > 0 {
		pkg := pkgs[0]
		for _, f := range pkg.Syntax {
			if fset.Position(f.Pos()).Filename == fileAbs {
				for _, c := range collectors {
					c.Collect(f, fset, fileAbs)
				}
				return f, fset, pkg.Types, pkg.TypesInfo, nil
			}
		}
	}

	// The file was not found in the loaded package syntax (e.g., excluded by
	// //go:build constraints in testdata fixtures). Fall back to direct parsing
	// and standalone type-checking, bypassing build constraints.
	src, typPkg, typInfo, err := parseAndTypeCheckDirect(fset, fileAbs)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if src != nil {
		for _, c := range collectors {
			c.Collect(src, fset, fileAbs)
		}
	}

	return src, fset, typPkg, typInfo, nil
}

// parseAndTypeCheckDirect parses a file directly (ignoring build constraints)
// and type-checks it as a standalone unit. Used as a fallback for files that
// are excluded from their package by build tags.
func parseAndTypeCheckDirect(fset *token.FileSet, fileAbs string) (*ast.File, *types.Package, *types.Info, error) {
	src, err := parser.ParseFile(fset, fileAbs, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not parse file %q: %v", fileAbs, err)
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	conf := &types.Config{
		Importer: importer.Default(),
		Error:    func(error) {}, // tolerate errors in isolated test fixtures
	}

	pkg, _ := conf.Check("", fset, []*ast.File{src}, info)

	return src, pkg, info, nil
}
