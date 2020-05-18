package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func outPath(p string) string { return strings.TrimSuffix(p, ".go") + ".out" }
func errPath(p string) string { return strings.TrimSuffix(p, ".go") + ".err" }

func equalBytes(t *testing.T, a, b []byte, normalize func([]byte) []byte) {
	t.Helper()
	if normalize != nil {
		a = normalize(a)
		b = normalize(b)
	}
	if !bytes.Equal(a, b) {
		t.Errorf(`bytes not equal
want: %s
got:  %s
`, a, b)
	}
}

func parseFlags(p string) {
	// Get the first line.
	b, err := ioutil.ReadFile(p)
	if err != nil {
		panic(fmt.Sprintf("failed to read file: %s", p))
	}
	idx := bytes.IndexByte(b, '\n')
	if idx == -1 {
		panic(fmt.Sprintf("no lines in file: %s", p))
	}
	// Does it have the prefix?
	const prefix = "//dedupimport"
	line := string(b[:idx])
	if !strings.HasPrefix(line, prefix) {
		return
	} else {
		line = strings.TrimPrefix(line, prefix)
	}
	// Parse.
	args := strings.Fields(line)
	for i := 0; i < len(args); {
		arg := args[i]
		switch arg {
		case "-keep":
			i++
			*strategy = args[i]
		case "-i":
			*importOnly = true
		default:
			panic("unhandled flag")
		}
		i++
	}
}

func resetFlags() {
	*strategy = "unnamed"
	*importOnly = false
}

func TestAll(t *testing.T) {
	fset := token.NewFileSet() // use the same fset
	filenames := []string{
		"testdata/cannot.go",
		"testdata/example.go",
		"testdata/named.go",
		"testdata/comment.go",
		"testdata/first1.go",
		"testdata/first2.go",
		"testdata/removed-comments.go",
		"testdata/plenty-imports.go",
		"testdata/dotimport.go",
		"testdata/space.go",
		"testdata/space-all1.go",
		"testdata/space-all2.go",
		"testdata/samename.go",
		"testdata/packagename.go",
		"testdata/scope1.go",
		"testdata/scope2.go",
		"testdata/misc.go",
		"testdata/invalid-ident.go",
		"testdata/import-only.go",
		"testdata/scopeafter1.go",
		"testdata/scopeafter2.go",
		"testdata/shortvar.go",
	}

	for _, path := range filenames {
		t.Run(path, func(t *testing.T) {
			resetFlags()
			parseFlags(path)
			runOneFile(t, fset, path)
		})
	}
}

func runOneFile(t *testing.T, fset *token.FileSet, path string) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	outContent, err := ioutil.ReadFile(outPath(path))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("failed to read out file: %s", err)
		}
	}

	errContent, err := ioutil.ReadFile(errPath(path))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("failed to read err file: %s", err)
		}
	}

	var outBuf, errBuf bytes.Buffer
	changedFile, err := processFile(fset, src, path)
	if err != nil {
		scanner.PrintError(&errBuf, err)
		equalBytes(t, errContent, errBuf.Bytes(), bytes.TrimSpace)
		return
	}

	if changedFile != nil {
		err = format.Node(&outBuf, fset, changedFile)
		if err != nil {
			t.Errorf("unexpected error formatting file: %s", err)
		}
		equalBytes(t, outContent, outBuf.Bytes(), bytes.TrimSpace)
	}
}

func TestGuessPackageName(t *testing.T) {
	type testcase struct {
		importPath string
		expect     string
	}
	testcases := []testcase{
		{"github.com/foo/bar", "bar"},
		{"github.com/foo/bar/v2", "bar"},
		{"github.com/foo/go-bar/v2", "bar"},
		{"github.com/foo/bar-go/v2", "bar"},
		{"gopkg.in/yaml.v2", "yaml"},
		{"gopkg.in/go-yaml.v2", "yaml"},
		{"gopkg.in/yaml-go.v2", "yaml"},
		{"github.com/nishanths/go-xkcd", "xkcd"},
		{"github.com/nishanths/lyft-go", "lyft"},
	}
	for _, tt := range testcases {
		t.Run(tt.importPath, func(t *testing.T) {
			got := guessPackageName(tt.importPath)
			if tt.expect != got {
				t.Errorf("expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}
