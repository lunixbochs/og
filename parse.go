package main

import (
	"github.com/lunixbochs/og/plugins"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
)

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
			plugins.ExpandTry(fset, f)
			bytes, err := plugins.CodeBytes(fset, f)
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

func ParseFile(filename string) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	plugins.ExpandTry(fset, f)
	bytes, err := plugins.CodeBytes(fset, f)
	if err != nil {
		log.Fatal(err)
	}
	return bytes
}
