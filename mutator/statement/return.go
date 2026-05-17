package statement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/jonbaldie/go-mutesting/v2/mutator"
)

func init() {
	mutator.Register("statement/return", MutatorReturnValue)
}

// MutatorReturnValue replaces each non-zero return value with the zero value
// for its type (false, 0, "", nil).
func MutatorReturnValue(_ *types.Package, info *types.Info, node ast.Node) []mutator.Mutation {
	n, ok := node.(*ast.ReturnStmt)
	if !ok || len(n.Results) == 0 || info == nil {
		return nil
	}

	var mutations []mutator.Mutation

	for i, result := range n.Results {
		t := info.TypeOf(result)
		if t == nil {
			continue
		}

		zero := zeroExprForType(t.Underlying())
		if zero == nil {
			continue
		}

		if isAlreadyZero(result) {
			continue
		}

		idx := i
		original := n.Results[idx]

		mutations = append(mutations, mutator.Mutation{
			Change: func() { n.Results[idx] = zero },
			Reset:  func() { n.Results[idx] = original },
		})
	}

	return mutations
}

// zeroExprForType returns the zero-value AST expression for the underlying type.
func zeroExprForType(t types.Type) ast.Expr {
	switch u := t.(type) {
	case *types.Basic:
		switch {
		case u.Kind() == types.Bool:
			return ast.NewIdent("false")
		case u.Info()&types.IsString != 0:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}
		case u.Info()&types.IsNumeric != 0:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		case u.Kind() == types.UnsafePointer:
			return ast.NewIdent("nil")
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Interface, *types.Signature:
		return ast.NewIdent("nil")
	case *types.Named:
		return zeroExprForType(u.Underlying())
	}
	return nil
}

// isAlreadyZero reports whether expr is already a zero-value literal,
// avoiding no-op mutations.
func isAlreadyZero(expr ast.Expr) bool {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Name == "nil" || n.Name == "false"
	case *ast.BasicLit:
		switch n.Kind {
		case token.INT, token.FLOAT:
			return n.Value == "0" || n.Value == "0.0"
		case token.STRING:
			return n.Value == `""`
		}
	}
	return false
}
