package main

import (
	"./plugins"
	"go/parser"
	"go/token"
	"log"
)

func Parse(filename string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("input:")
	plugins.PrintCode(fset, f)
	plugins.ExpandTry(fset, f)
	log.Print("output:")
	plugins.PrintCode(fset, f)
}
