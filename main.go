package main

import (
	"fmt"
	"os"
)

func SplitArgs(args []string) (string, []string) {
	if len(args) < 2 {
		return "", nil
	}
	return args[1], args[2:]
}

func main() {
	cmd, args := SplitArgs(os.Args)
	og := NewOg(args, ".")
	if cmd == "" {
		cmd = "help"
	}

	switch cmd {
	case "build":
		og.Build()
	case "help":
		og.Help()
	case "parse":
		if len(args) < 1 {
			fmt.Println("Usage: og parse <filename>")
			os.Exit(1)
		}
		ParseFile(args[0])
	default:
		og.Default(cmd)
	}
}
