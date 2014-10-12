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

func next(int n) (int, error) {
	return n, nil
}

func call(n int) (int, error) {
	fmt.Println(n)
	try(next(-1))
	output := try(next(-1), FATAL)
	output = try(next(-1), FATAL, "message")
	return output, nil
}

func main() {
	fmt.Println("hi")
	output := try(call(1), FATAL, "message")
	output = try(call(2), "message")
	try(call(3), func(err error) {fmt.Println(err)})
	try(call(4), RETURN)
	try(call(5))
	try(call(6), func() {return 1})
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
