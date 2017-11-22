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

	res := dedupe(file.Imports)

	resLookup := make(map[*ast.ImportSpec]struct{}, len(res))
	for _, im := range res {
		resLookup[im] = struct{}{}
	}

	for i := range file.Decls {
		// we only care about import declarations: they have the type GenDecl.
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
			if _, ok := resLookup[im]; ok {
				// was not removed during deduping.
				trimmed = append(trimmed, spec)
			}
		}
		genDecl.Specs = trimmed
		file.Decls[i] = genDecl
		// format.Node(os.Stdout, fset, file)
	}

	var nonEmptyDecls []ast.Decl
	for _, decl := range file.Decls {
		// we only care about import declarations: they have the type GenDecl.
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		if len(genDecl.Specs) != 0 {
			nonEmptyDecls = append(nonEmptyDecls, decl)
		}
	}
	file.Decls = nonEmptyDecls

	// ast.Print(fset, file.Decls)
	format.Node(os.Stdout, fset, file)

	// fmt.Println(file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ImportSpec))
	// fmt.Println(file.Imports[0])

	// ast.Print(fset, file.Decls)
	// TODO: need to associate back with GenDecl Tok=import

	// ast.SortImports(fset, file)
	// format.Node(os.Stdout, fset, file)
	// ast.Print(fset, file)
	// fmt.Println(file.Scope.Lookup("foo"))
	// fmt.Println(file.Scope.Lookup("f"))
	// fmt.Println(file.Scope.Lookup("p"))

	return nil
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
		// NOTE: the panics below indicate conditions that should have been
		// caught already by the parser.
		if spec.Path.Kind != token.STRING {
			panicf("import path %s is not a string", spec.Path.Value)
		}
		// skip dot imports. for now, let's assume it's okay to have both a
		// dot import and a regular import.
		if spec.Name != nil && spec.Name.Name == "." {
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
	s := fmt.Sprintf(format, v)
	panic(s)
}
