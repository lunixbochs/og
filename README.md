`og`: Language and tool enhancements for Go
----

Installation
----

    go get github.com/lunixbochs/og

You can update og to the latest version with `og update`.

Usage
----

`og` is a `go` frontend with additional features, such as a code preprocessor.

Use it as you would the existing `go` command, like `go build` or `go run`.

It also provides the following extra commands:

    og help
    ...
        gen         generate preprocessed source tree
        parse       preprocess one source file
        update      update og command
    ...

Features
----

 - Preprocessor
    - Easily perform AST transformations on Go projects as they are built
    - Safe: will never modify or overwrite your existing files
    - Fast: adds almost no time to your project build
 - Language Extensions
    - `try()`: reduces the `if err != nil {}` pattern to a single line.

            // before
            tmp, err := call()
            if err != nil {
                return nil, err
            }
            // after
            tmp := try(call())
