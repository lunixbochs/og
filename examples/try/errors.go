package main

func main() {
	try := "this will fail, as try is a reserved keyword"
	// unsupported behavior:
	switch test {
	case try():
	}
	select {
	case try(<-channel):
	}
	for i := range try() {

	}
	for i := 0; i < try(dangerous(list)); i++ {

	}
	// this will be fun
	if (try(a) || try(b)) && try(c) {

	}
	return try(call())
}
