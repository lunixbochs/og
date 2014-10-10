package main

import (
	"go/parser"
	"go/token"
	"log"
)

func main() {
	src := `
package main

import (
	"fmt"
	"log"
)

func call(n int) (int, error) {
	try(call())
	fmt.Println(n)
	return n, nil
}

func main() {
	fmt.Println("hi")
	output := try(call(1), FATAL, "message")
	output = try(call(2), "message")
	output = try(call(3), func(err error) {fmt.Println(err)})
	output = try(call(4))
	try(call(5))
	fmt.Println("end", output)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("input:")
	PrintCode(fset, f)
	ExpandTry(fset, f)
	log.Print("output:")
	PrintCode(fset, f)
}
