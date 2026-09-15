package astutil

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"testing"
)

func TestZeroExprForType_UnexportedStructInImportedPackage(t *testing.T) {
	dependency := `package dep

type secret struct{ Val int }

func New() secret { return secret{1} }
`
	source := `package example

import "example.com/dep"

func Get() any {
	return dep.New()
}
`
	assertCrossPackageZeroExpr(t, dependency, source, "")
}

func TestZeroExprForType_ExportedStructInImportedPackage(t *testing.T) {
	dependency := `package dep

type Public struct{ Val int }

func New() Public { return Public{1} }
`
	source := `package example

import "example.com/dep"

func Get() any {
	return dep.New()
}
`
	assertCrossPackageZeroExpr(t, dependency, source, "dep.Public{}")
}

// assertCrossPackageZeroExpr type-checks dependency as "example.com/dep",
// then source as a package importing it, and compares the zero expression
// built for the first return result against want. An empty want means the
// zero expression must be nil.
func assertCrossPackageZeroExpr(t *testing.T, dependency, source, want string) {
	t.Helper()

	fset := token.NewFileSet()
	depFile, err := parser.ParseFile(fset, "dep.go", dependency, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse dependency: %v\n%s", err, dependency)
	}
	depPkg, err := (&types.Config{Importer: importer.Default()}).Check("example.com/dep", fset, []*ast.File{depFile}, nil)
	if err != nil {
		t.Fatalf("type-check dependency: %v\n%s", err, dependency)
	}

	file, err := parser.ParseFile(fset, "example.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v\n%s", err, source)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	config := &types.Config{Importer: fixedImporter{pkg: depPkg}}
	pkg, err := config.Check("example.com/example", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("type-check source: %v\n%s", err, source)
	}

	ret := firstReturnStmt(t, file)
	typ := info.TypeOf(ret.Results[0])
	if typ == nil {
		t.Fatal("no type for return expression")
	}

	got := ZeroExprForType(typ, pkg)
	if want == "" {
		if got != nil {
			t.Fatalf("ZeroExprForType(%s) = %s, want nil", typ, printZeroNode(t, got))
		}
		return
	}
	if got == nil {
		t.Fatalf("ZeroExprForType(%s) = nil, want %s", typ, want)
	}
	if printed := printZeroNode(t, got); printed != want {
		t.Fatalf("ZeroExprForType(%s) = %s, want %s", typ, printed, want)
	}
}

// fixedImporter resolves every import path to the same package, which is
// enough for the two-package sources used in these tests.
type fixedImporter struct {
	pkg *types.Package
}

func (i fixedImporter) Import(path string) (*types.Package, error) {
	if path == i.pkg.Path() {
		return i.pkg, nil
	}
	return importer.Default().Import(path)
}

func printZeroNode(t *testing.T, node ast.Node) string {
	t.Helper()

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), node); err != nil {
		t.Fatalf("print node: %v", err)
	}
	return buf.String()
}

func firstReturnStmt(t *testing.T, file *ast.File) *ast.ReturnStmt {
	t.Helper()

	var found *ast.ReturnStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if found != nil {
			return false
		}
		ret, ok := node.(*ast.ReturnStmt)
		if ok {
			found = ret
			return false
		}
		return true
	})
	if found == nil {
		t.Fatal("source contains no return statement")
	}
	return found
}
