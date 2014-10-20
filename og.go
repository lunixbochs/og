package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
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
	if err != nil {
		o.Exit(err, out)
	}
	work, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	// TODO: respect flag to not remove working directory
	dirSearch := regexp.MustCompile(`# _(.+)`)
	for _, sub := range dirSearch.FindAllSubmatch(out, -1) {
		src := string(sub[1])
		dst := path.Join(work, "_", src)
		err := os.MkdirAll(dst, os.ModeDir|0700)
		if err != nil {
			os.RemoveAll(work)
			log.Fatal(err)
		}
		err = ParseDir(src, dst)
		if err != nil {
			os.RemoveAll(work)
			log.Fatal(err)
		}
		// TODO: what do ./*.go lines look like with `go build -n` on Windows?
		// TODO: need to replace filenames more cleanly
		// at least group commands by chdir
		goLineSearch := regexp.MustCompile(`(?m)^.+?(\./.+?\.go).*?$`)
		next := goLineSearch.FindIndex(out)
		line := out[next[0]:next[1]]
		goSearch := regexp.MustCompile(`\./(.+\.go)`)
		for _, f := range goSearch.FindAllIndex(line, -1) {
			name := string(line[f[0]:f[1]])
			repl := path.Join("$WORK", "_", src, path.Base(name))
			out = bytes.Replace(out, []byte(name), []byte(repl), 1)
		}
	}
	env := os.Environ()
	env = append(env, "WORK="+work)
	lines := bytes.Split(out, []byte("\n"))
	fmt.Println(work)
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			// TODO: Windows support
			fmt.Println(string(line))
			cmd := exec.Command("sh", "-c", string(line))
			cmd.Env = env
			out, _ := cmd.CombinedOutput()
			fmt.Printf("> %s\n", out)
		}
	}
	os.RemoveAll(work)
	o.Exit(nil, nil)
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
