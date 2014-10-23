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

func (o *Og) Default(cmd string) ([]byte, int) {
	out, err := o.Exec(cmd)
	return out, exitStatus(err)
}

func (o *Og) Dispatch(cmd string) int {
	var out []byte
	var code int
	switch cmd {
	case "build":
		out, code = o.CmdBuild()
	case "gen":
		code = o.CmdGen()
	case "help":
		out, code = o.CmdHelp()
	case "parse":
		out, code = o.CmdParse()
	case "run":
		out, code = o.CmdRun()
	case "update":
		out, code = o.CmdUpdate()
	default:
		out, code = o.Default(cmd)
	}
	fmt.Printf("%s", out)
	return code
}

func (o *Og) Exec(cmd string, args ...string) ([]byte, error) {
	args = append(append([]string{cmd}, args...), o.Args...)
	return exec.Command("go", args...).CombinedOutput()
}

func (o *Og) RelPath(path string) string {
	if strings.HasPrefix(path, o.Dir) {
		return strings.Replace(path, o.Dir, "", 1)
	}
	return path
}

func (o *Og) GenFiles(outdir string, names ...string) error {
	for _, name := range names {
		dst := path.Join(outdir, o.RelPath(name))
		err := os.MkdirAll(path.Dir(dst), os.ModeDir|0700)
		if err != nil {
			return err
		}
		out, err := ParseFile(name)
		if err != nil {
			return err
		}
		ioutil.WriteFile(dst, out, 0600)
	}
	return nil
}

func (o *Og) Gen(outdir string) error {
	out, err := exec.Command("go", "build", "-n").CombinedOutput()
	if err != nil {
		fmt.Printf("%s", out)
		return err
	}
	// TODO: respect flag to not remove working directory
	dirSearch := regexp.MustCompile(`# _(.+)`)
	for _, sub := range dirSearch.FindAllSubmatch(out, -1) {
		src := string(sub[1])
		dst := path.Join(outdir, o.RelPath(src))
		err := os.MkdirAll(dst, os.ModeDir|0700)
		if err != nil {
			return err
		}
		err = ParseDir(src, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Og) CmdBuild() ([]byte, int) {
	steps, err := o.ParseBuild("build", o.Args...)
	if err != nil {
		log.Fatal(err)
	}
	err = o.RewriteBuild(steps)
	if err != nil {
		log.Fatal(err)
	}
	err = o.RunBuild(steps)
	if err != nil {
		log.Fatal(err)
	}
	return nil, 0
}

func (o *Og) CmdGen() int {
	if len(o.Args) < 1 {
		fmt.Println("Usage: og gen <output dir>")
		return 1
	}
	err := o.Gen(o.Args[0])
	if err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func (o *Og) CmdHelp() ([]byte, int) {
	if len(o.Args) > 0 {
		// TODO: add help shims for gen, parse
		return o.Default("help")
	}
	modifyHelp := func(help []byte, helps []string) []byte {
		search := regexp.MustCompile(`(?m)^.+?commands.+$`)
		idx := search.FindIndex(help)
		left, right := help[:idx[0]], help[idx[1]:]
		template := "%sOg commands:\n\n    %s\n\nGo commands:%s"
		tmp := fmt.Sprintf(template, left, strings.Join(helps, "\n    "), right)
		return []byte(tmp)
	}
	out, _ := exec.Command("go").CombinedOutput()
	out = bytes.Replace(out, []byte("Go"), []byte("Og"), 1)
	out = bytes.Replace(out, []byte("go command"), []byte("og command"), 1)
	out = bytes.Replace(out, []byte("go help"), []byte("og help"), -1)
	helps := []string{
		"gen         generate preprocessed source tree",
		"parse       preprocess one source file",
		"update      update og command",
	}
	out = modifyHelp(out, helps)
	return out, 0
}

func (o *Og) CmdParse() ([]byte, int) {
	if len(o.Args) < 1 {
		return []byte("Usage: og parse <filename>"), 1
	}
	out, err := ParseFile(o.Args[0])
	if err != nil {
		log.Fatal(err)
	}
	return out, 0
}

func (o *Og) CmdRun() ([]byte, int) {
	if len(o.Args) < 1 {
		out, err := exec.Command("go", "help", "run").CombinedOutput()
		return out, exitStatus(err)
	}
	steps, err := o.ParseBuild("run", o.Args...)
	if err != nil {
		log.Fatal(err)
	}
	o.RewriteBuild(steps)

	return nil, 0
}

func (o *Og) CmdUpdate() ([]byte, int) {
	// TODO: don't hardcode URL?
	out, err := exec.Command("go", "get", "-u", "github.com/lunixbochs/og").CombinedOutput()
	return out, exitStatus(err)
}
