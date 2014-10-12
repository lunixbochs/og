package main

import (
	"./plugins"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./og <source filename.go>")
		fmt.Println("  Example: ./og examples/try/basic.go")
		return
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, os.Args[1], nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("input:")
	plugins.PrintCode(fset, f)
	plugins.ExpandTry(fset, f)
	log.Print("output:")
	plugins.PrintCode(fset, f)
}
