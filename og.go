package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

type Og struct {
	Dir  string
	Args []string
}

type BuildStep struct {
	Dir  string
	Cmds [][]byte
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

func (o *Og) ParseBuild() ([]*BuildStep, error) {
	out, err := exec.Command("go", "build", "-n").CombinedOutput()
	if err != nil {
		fmt.Printf("%s", out)
		return nil, err
	}
	prefixRe := regexp.MustCompile(`(?m)^# _(.+)$`)
	var steps []*BuildStep
	var prefix string
	var cur [][]byte
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("#")) {
			tmp := prefixRe.FindSubmatch(line)
			if len(tmp) > 0 {
				if prefix != "" && len(cur) > 0 {
					steps = append(steps, &BuildStep{prefix, cur})
					cur = nil
				}
				prefix = string(tmp[1])
			}
		} else {
			cur = append(cur, line)
		}
	}
	if len(cur) > 0 {
		steps = append(steps, &BuildStep{prefix, cur})
	}
	return steps, nil
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
		// TODO: collapse this path, probably when I redo the `go build` parsing
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
	steps, err := o.ParseBuild()
	if err != nil {
		log.Fatal(err)
	}
	work, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	err = o.Gen(work)
	if err != nil {
		log.Fatal(err)
		os.RemoveAll(work)
	}
	goLine := regexp.MustCompile(`(?m)^.+?(\./.+?\.go).*?$`)
	goNuke := regexp.MustCompile(`(?m)\./.+$`)
	// TODO: what do ./*.go lines look like with `go build -n` on Windows?
	for _, step := range steps {
		prefix := step.Dir
		for i, cmd := range step.Cmds {
			if !goLine.Match(cmd) {
				continue
			}
			files, err := filepath.Glob(path.Join(prefix, "*.go"))
			if err != nil {
				log.Fatal(err)
			}
			for i, name := range files {
				// TODO: instead of escaping, exec commands without a shell
				name = strings.Replace(name, o.Dir, "", 1)
				files[i] = "\"" + path.Join("$WORK", escapeFilename(name)) + "\""
			}
			suffix := " " + strings.Join(files, " ")
			step.Cmds[i] = append(goNuke.ReplaceAll(cmd, []byte(" ")), []byte(suffix)...)
		}
	}
	// TODO: support `og build -n` right here (and make sure to skip the gen step)
	env := os.Environ()
	env = append(env, "WORK="+work)
	for _, step := range steps {
		for _, line := range step.Cmds {
			cmd := exec.Command("sh", "-c", string(line))
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			out = bytes.TrimSpace(out)
			if err != nil && !bytes.HasPrefix(line, []byte("mkdir")) {
				return out, exitStatus(err)
			}
			if len(out) > 0 {
				fmt.Printf("%s\n", out)
			}
		}
	}
	os.RemoveAll(work)
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
	}
	out = modifyHelp(out, helps)
	return out, 0
}

func (o *Og) CmdParse() ([]byte, int) {
	if len(o.Args) < 1 {
		return []byte("Usage: og parse <filename>"), 1
	}
	return ParseFile(o.Args[0]), 0
}
