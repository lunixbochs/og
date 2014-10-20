package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
	dirSearch := regexp.MustCompile(`# _(.+)`)
	for _, path := range dirSearch.FindAllSubmatch(out, -1) {
		fmt.Println(string(path[1]))
	}
	o.Exit(err, out)
}

func (o *Og) Help() {
	if len(o.Args) > 0 {
		// TODO: add help shims for gen, parse
		o.Default("help")
		return
	}
	modifyHelp := func(help []byte, helps []string) []byte {
		search := regexp.MustCompile(`(?m)^.+?commands.+$`)
		idx := search.FindIndex(help)
		left, right := help[:idx[0]], help[idx[1]:]
		template := "%sOg commands:\n\n    %s\n\nGo commands:%s"
		tmp := fmt.Sprintf(template, left, strings.Join(helps, "\n    "), right)
		return []byte(tmp)
	}
	out, err := exec.Command("go").CombinedOutput()
	out = bytes.Replace(out, []byte("Go"), []byte("Og"), 1)
	out = bytes.Replace(out, []byte("go command"), []byte("og command"), 1)
	out = bytes.Replace(out, []byte("go help"), []byte("og help"), -1)
	helps := []string{
		"gen         generate preprocessed source tree",
		"parse       preprocess one source file",
	}
	out = modifyHelp(out, helps)
	o.Exit(err, out)
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
