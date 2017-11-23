package main

import (
	"bytes"
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

func TestAll(t *testing.T) {
	fset := token.NewFileSet() // use the same fset
	filenames := []string{"testdata/cannot.go", "testdata/example.go"}

	for i, path := range filenames {
		if testing.Verbose() {
			t.Logf("testing file [%d]: %s", i, path)
		}
		runOneFile(t, fset, path)
	}
}

func runOneFile(t *testing.T, fset *token.FileSet, path string) {
	t.Helper()
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
