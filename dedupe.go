package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

const help = `usage: dupeimports [flags] [path ...]`

func usage() {
	fmt.Fprintf(os.Stderr, "%s\n", help)
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	diff       = flag.Bool("d", false, "display diff instead of rewriting files")
	allErrors  = flag.Bool("e", false, "report all parse errors, not just the first 10 on different lines")
	list       = flag.Bool("l", false, "list files with duplicate import")
	rewriteSrc = flag.Bool("w", false, "write result to source file instead of stdout")
	strategy   = flag.String("s", "unnamed", "which import to keep; one of: first, named, unnamed")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("dupeimports: ")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		if *rewriteSrc {
			log.Print("cannot use -w with stdin")
			os.Exit(2)
		}
		err := processFile(os.Stdin, os.Stdout, "", true)
		if err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	} else {
		// TODO
	}
}

func parserMode() parser.Mode {
	if *allErrors {
		return parser.ParseComments | parser.AllErrors
	}
	return parser.ParseComments
}

type Scope struct {
	// TODO: keep a fileset here for better positioning?
	node   ast.Node
	outer  *Scope
	inner  []*Scope
	idents map[string]*ast.Ident
	done   bool // completed "parsing" this scope?
}

func newScope(node ast.Node) *Scope {
	sc := new(Scope)
	sc.node = node
	return sc
}

func (sc *Scope) assertDone() {
	if !sc.done {
		panicf("scope not done")
	}
}

func (sc *Scope) markDone() {
	if sc.done {
		panicf("scope already done")
	}
	sc.done = true
}

func (sc *Scope) addIdent(ident *ast.Ident) {
	if sc.idents == nil {
		sc.idents = make(map[string]*ast.Ident)
	}
	sc.idents[ident.Name] = ident
}

// delcared returns whether the named identifier
// is declared in this scope.
func (sc *Scope) declared(name string) bool {
	if sc == nil {
		return false
	}
	sc.assertDone()
	_, ok := sc.idents[name]
	return ok
}

// available returns whether the named identifier is
// delcared in this scope or any of the outer scopes.
func (sc *Scope) available(name string) bool {
	if sc == nil {
		return false
	}
	for c := sc; c != nil; c = sc.outer {
		if c.declared(name) {
			return true
		}
	}
	return false
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
			return false
		case *ast.TypeSpec:
			cur.addIdent(x.Name)
			return false
		case *ast.FuncDecl:
			cur.addIdent(x.Name)
			inner := walkFuncDecl(x)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
			return false
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

// walkFuncLit is similar to walkFuncDecl expect that it doesn't have
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
			return false
		case *ast.FuncLit:
			// unlike a FuncDecl, a FuncLit has no name,
			// so there's nothing to ident to add to cur.
			inner := walkFuncLit(xx)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
			return false
		case *ast.TypeSpec:
			cur.addIdent(xx.Name)
			return false
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
			return false
		case *ast.BlockStmt:
			if x == xx {
				// Skip original argument to Inspect.
				// It should have been handled by the caller.
				return true
			}
			inner := walkBlockStmt(xx)
			cur.inner = append(cur.inner, inner)
			inner.outer = cur
			return false
		}
		return true
	})

	cur.markDone()
	return cur
}

// file set for the command invocation.
var fset = token.NewFileSet()

func processFile(in io.Reader, out io.Writer, filename string, stdin bool) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	file, err := parser.ParseFile(fset, filename, src, parserMode())
	if err != nil {
		return err
	}

	// ast.Print(fset, file)
	file.Imports = dedupe(file.Imports)
	trimImportDecls(file)

	scope := walkFile(file)
	_ = scope

	format.Node(os.Stdout, fset, file)
	// ast.SortImports(fset, file)
	// format.Node(os.Stdout, fset, file)

	return nil
}

// trimImportDecls trims the file's import declarations based on the import
// specifications still present in file.Imports.
func trimImportDecls(file *ast.File) {
	lookup := make(map[*ast.ImportSpec]struct{}, len(file.Imports))
	for _, im := range file.Imports {
		lookup[im] = struct{}{}
	}

	for i := range file.Decls {
		genDecl, ok := file.Decls[i].(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		var trimmed []ast.Spec
		for _, spec := range genDecl.Specs {
			im, ok := spec.(*ast.ImportSpec)
			if !ok {
				// WTF, doesn't match godoc
				panicf("expected ImportSpec")
			}
			if _, ok := lookup[im]; ok {
				// was not removed during deduping.
				trimmed = append(trimmed, spec)
			}
		}
		genDecl.Specs = trimmed
		file.Decls[i] = genDecl
	}

	var nonEmptyDecls []ast.Decl
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			nonEmptyDecls = append(nonEmptyDecls, decl)
			continue
		}
		if len(genDecl.Specs) != 0 {
			nonEmptyDecls = append(nonEmptyDecls, decl)
		}
	}
	file.Decls = nonEmptyDecls
}

// dedupe removes duplicate imports.
// The input slice is never modified.
func dedupe(input []*ast.ImportSpec) []*ast.ImportSpec {
	imports := make([]*ImportSpec, len(input))
	for i := range input {
		imports[i] = &ImportSpec{input[i], false}
	}

	importPaths := make(map[string][]*ImportSpec)
	for _, im := range imports {
		spec := im.spec
		// NOTE: The panics below indicate conditions that should have been
		// caught already by the parser.
		if spec.Path.Kind != token.STRING {
			panicf("import path %s is not a string", spec.Path.Value)
		}
		// skip dot and side effect imports. for now, let's assume it's okay
		// to have both these coexist with regular imports. In fact, it looks
		// like it's necessary to not remove _; that's the only way both _
		// and regular import can be used together in a file.
		if spec.Name != nil && (spec.Name.Name == "." || spec.Name.Name == "_") {
			continue
		}
		// normalize `fmt` vs. "fmt", for instance
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			panicf("unquoting path: %s", err)
		}
		importPaths[path] = append(importPaths[path], im)
	}

	duplicateImportPaths := make(map[string][]*ImportSpec)
	for p, v := range importPaths {
		if len(v) > 1 {
			duplicateImportPaths[p] = v
		}
	}

	for _, v := range duplicateImportPaths {
		switch *strategy {
		case "unnamed":
			idx := -1
			for i := range v {
				if v[i].spec.Name == nil {
					idx = i
					break
				}
			}
			keepIdx := idx
			if keepIdx == -1 {
				keepIdx = 0
			}
			for i := 0; i < len(v); i++ {
				if i != keepIdx {
					v[i].remove = true
				}
			}
		case "first":
			for i := 1; i < len(v); i++ {
				v[i].remove = true
			}
		case "named":
			idx := -1
			length := -1
			for i := range v {
				if v[i].spec.Name != nil && (len(v[i].spec.Name.Name) < length || length == -1) {
					idx = i
					length = len(v[i].spec.Name.Name)
				}
			}
			keepIdx := idx
			if keepIdx == -1 {
				keepIdx = 0
			}
			for i := 0; i < len(v); i++ {
				if i != keepIdx {
					v[i].remove = true
				}
			}
		}
	}

	var res []*ast.ImportSpec
	for _, im := range imports {
		if !im.remove {
			res = append(res, im.spec)
		}
	}
	return res
}

type ImportSpec struct {
	spec   *ast.ImportSpec
	remove bool
}

func removeDecl(a []ast.Decl, i int) []ast.Decl {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = nil
	a = a[:len(a)-1]
	return a
}

func removeImportSpec(a []*ast.ImportSpec, i int) []*ast.ImportSpec {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = nil
	a = a[:len(a)-1]
	return a
}

func panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	panic(s)
}

func debugf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}
