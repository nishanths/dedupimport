package main

import (
	"go/ast"
	"go/token"
)

type Scope struct {
	node   ast.Node              // the underlying node that defines this scope (*ast.File, *ast.FuncDecl, *ast.BlockStmt, *ast.FuncLit)
	outer  *Scope                // parent scope, or nil
	inner  []*Scope              // immediate inner scopes
	idents map[string]*ast.Ident // idents in this scope; the key is the name of the ident for fast lookup
	done   bool                  // completed "parsing" this scope; exists to guard against programmer error
}

func newScope(node ast.Node) *Scope {
	sc := new(Scope)
	sc.node = node
	return sc
}

func (sc *Scope) assertDone() {
	if !sc.done {
		panic("scope not done")
	}
}

func (sc *Scope) markDone() {
	if sc.done {
		panic("scope already done")
	}
	sc.done = true
}

func (sc *Scope) addIdent(ident *ast.Ident) {
	if sc.idents == nil {
		sc.idents = make(map[string]*ast.Ident)
	}
	sc.idents[ident.Name] = ident
}

// declared returns whether the named identifier
// is declared in this scope.
func (sc *Scope) declared(name string) bool {
	sc.assertDone()
	_, ok := sc.idents[name]
	return ok
}

// available returns whether the named identifier is
// delcared in this scope or any of the outer scopes.
func (sc *Scope) available(name string) bool {
	sc.assertDone()
	for c := sc; c != nil; c = c.outer {
		if c.declared(name) {
			return true
		}
	}
	return false
}

// traverse walks the scope in breadth first order.
func (sc *Scope) traverse(fn func(*Scope) bool) {
	q := []*Scope{sc}
	for len(q) != 0 {
		var c *Scope
		c, q = q[0], q[1:]
		if !fn(c) {
			break
		}
		for _, in := range c.inner {
			q = append(q, in)
		}
	}
}

// Notes
// -----
//
// https://golang.org/ref/spec#Declarations_and_scope
// Go is lexically scoped using blocks:
// 1. The scope of a predeclared identifier is the universe block.
// 2. The scope of an identifier denoting a constant, type, variable, or
//    function (but not method) declared at top level (outside any function) is
//    the package block.
// 3. The scope of the package name of an imported package is the file block
//    of the file containing the import declaration.
// 4. The scope of an identifier denoting a method receiver, function
//    parameter, or result variable is the function body.
// 5. The scope of a constant or variable identifier declared inside a
//    function begins at the end of the ConstSpec or VarSpec (ShortVarDecl for
//    short variable declarations) and ends at the end of the innermost
//    containing block.
// 6. The scope of a type identifier declared inside a function begins at the
//    identifier in the TypeSpec and ends at the end of the innermost containing
//    block.

func walkFile(file *ast.File) *Scope {
	cur := newScope(file)

	ast.Inspect(file, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.ValueSpec:
			for _, name := range x.Names {
				cur.addIdent(name)
			}
		case *ast.TypeSpec:
			cur.addIdent(x.Name)
		case *ast.FuncDecl:
			cur.addIdent(x.Name)
			inner := walkFuncDecl(x)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
		case *ast.FuncLit:
			inner := walkFuncLit(x)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
		}
		return true
	})

	cur.markDone()
	return cur
}

func walkFuncDecl(x *ast.FuncDecl) *Scope {
	cur := newScope(x)

	// add receivers idents
	if x.Recv != nil {
		for _, field := range x.Recv.List {
			for _, name := range field.Names {
				cur.addIdent(name)
			}
		}
	}
	// add params idents
	for _, field := range x.Type.Params.List {
		for _, name := range field.Names {
			cur.addIdent(name)
		}
	}
	// add returns idents
	if x.Type.Results != nil {
		for _, field := range x.Type.Results.List {
			for _, name := range field.Names {
				cur.addIdent(name)
			}
		}
	}
	// walk the body
	if x.Body != nil {
		blockScope := walkBlockStmt(x.Body)
		cur.inner = append(cur.inner, blockScope)
		blockScope.outer = cur
	}

	cur.markDone()
	return cur
}

// walkFuncLit is similar to walkFuncDecl expect that a FuncLit doesn't have
// receivers.
func walkFuncLit(x *ast.FuncLit) *Scope {
	cur := newScope(x)

	// add params idents
	for _, field := range x.Type.Params.List {
		for _, name := range field.Names {
			cur.addIdent(name)
		}
	}
	// add returns idents
	if x.Type.Results != nil {
		for _, field := range x.Type.Results.List {
			for _, name := range field.Names {
				cur.addIdent(name)
			}
		}
	}
	// walk the body
	if x.Body != nil {
		blockScope := walkBlockStmt(x.Body)
		cur.inner = append(cur.inner, blockScope)
		blockScope.outer = cur
	}

	cur.markDone()
	return cur
}

func walkBlockStmt(x *ast.BlockStmt) *Scope {
	cur := newScope(x)

	ast.Inspect(x, func(node ast.Node) bool {
		switch xx := node.(type) {
		case *ast.ValueSpec:
			for _, name := range xx.Names {
				cur.addIdent(name)
			}
		case *ast.FuncLit:
			// unlike a FuncDecl, a FuncLit has no name,
			// so there's no ident to add to cur.
			inner := walkFuncLit(xx)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
		case *ast.TypeSpec:
			cur.addIdent(xx.Name)
		case *ast.AssignStmt:
			// The Lhs contains the identifier.  We only care about short
			// variable declarations, which use token.DEFINE.
			if xx.Tok == token.DEFINE {
				for _, expr := range xx.Lhs {
					if ident, ok := expr.(*ast.Ident); ok {
						cur.addIdent(ident)
					}
				}
			}
		case *ast.BlockStmt:
			if x == xx {
				// Skip original argument to Inspect.
				// It should have been handled by the caller.
				// TODO: feels hacky? find a better place for this.
				return true
			}
			inner := walkBlockStmt(xx)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
		}
		return true
	})

	cur.markDone()
	return cur
}
