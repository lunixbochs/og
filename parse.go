package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
)

func ParseAst(fset *token.FileSet, f *ast.File) {
	ParseTry(fset, f)
}

func ParseDir(src, dst string) error {
	_, err := os.Stat(dst)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, src, nil, 0)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			ParseAst(fset, f)
			bytes, err := CodeBytes(fset, f)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(path.Join(dst, path.Base(fname)), bytes, 0700)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ParseFile(filename string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	ParseAst(fset, f)
	bytes, err := CodeBytes(fset, f)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
