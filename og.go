package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
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

func (o *Og) Exec(cmd string, args ...string) ([]byte, error) {
	args = append(append([]string{cmd}, args...), o.Args...)
	return exec.Command("go", args...).CombinedOutput()
}

func (o *Og) Build() {
	out, err := o.Exec("build", "-n")
	o.Exit(err, out)
}

func (o *Og) Help() {
	out, err := exec.Command("go").CombinedOutput()
	out = bytes.Replace(out, []byte("Go"), []byte("Og"), 1)
	out = bytes.Replace(out, []byte("go command"), []byte("og command"), 1)
	out = bytes.Replace(out, []byte("go help"), []byte("og help"), -1)
	goGet := regexp.MustCompile(`\s+get`)
	idx := goGet.FindIndex(out)
	genInsert := []byte("\n    gen         generate preprocessed source tree")
	o.Exit(err, out[:idx[0]], genInsert, out[idx[0]:])
}

func (o *Og) Default(cmd string) {
	out, err := o.Exec(cmd)
	o.Exit(err, out)
}

func (o *Og) Exit(err error, output ...[]byte) {
	for _, v := range output {
		fmt.Printf("%s", v)
	}
	if err == nil {
		os.Exit(0)
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		}
	}
	log.Fatal("syscall.WaitStatus doesn't seem to be supported on your platform")
}
