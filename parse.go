package main

import (
	"./plugins"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func ParseFile(filename string) {
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
	fmt.Printf("%s\n", bytes)
	os.Exit(0)
}
