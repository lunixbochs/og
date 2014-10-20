package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"syscall"
)

type Og struct {
	Dir  string
	Args []string
}

func NewOg(args []string, dir string) *Og {
	var err error
	if dir == "" || dir == "." {
		dir, err = syscall.Getwd()
		if err != nil {
			log.Fatal("failed to get working directory:", err)
		}
	}
	if _, err := exec.LookPath("go"); err != nil {
		log.Fatal("could not find `go` command:", err)
	}
	return &Og{dir, args}
}

func (o *Og) Build() {
	args := append([]string{"-n"}, o.Args...)
	exec.Command("go", args...)
}

func (o Og) Usage() {
	out, _ := exec.Command("go").CombinedOutput()
	out = bytes.Replace(out, []byte("Go"), []byte("Og"), 1)
	out = bytes.Replace(out, []byte("go command"), []byte("og command"), 1)
	out = bytes.Replace(out, []byte("go help"), []byte("og help"), -1)
	goGet := regexp.MustCompile(`\s+get`)
	idx := goGet.FindIndex(out)
	genInsert := "\n    gen         generate preprocessed source tree"
	fmt.Printf("%s%s%s", out[:idx[0]], genInsert, out[idx[0]:])
}
