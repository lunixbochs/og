package main

import (
	"fmt"
	"log"
)

func next(n int) (int, error) {
	return n, nil
}

func call(n int) (int, error) {
	fmt.Println(n)
	output := try(next(-1), FATAL)
	output = try(next(-1), FATAL, "message")
	// doesn't work yet, because we don't discard the first return var
	//	try(next(-1), FATAL, "message")
	return output, nil
}

func bareCall(n int) error {
	_, err := call(n)
	return err
}

func main() {
	fmt.Println("hi")
	output := try(call(1), FATAL, "message")
	output = try(call(2), "message")
	try(bareCall(3), func() { fmt.Println(err) })
	try(bareCall(4), RETURN)
	try(bareCall(5))
	try(bareCall(6), func() { return })
	fmt.Println("end", output)
}
