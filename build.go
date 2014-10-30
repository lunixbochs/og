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
)

type BuildStep struct {
	Scope string
	Dir   string
	Cmds  []string
	Pure  []string
}

func (step *BuildStep) Init() {
	step.Pure = make([]string, len(step.Cmds))
	copy(step.Pure, step.Cmds)
}

var goLine *regexp.Regexp = regexp.MustCompile(`(?m)^.+?(\./.+?\.go).*?$`)
var goFiles *regexp.Regexp = regexp.MustCompile(`(?m)\./.+$`)

func GetCmdFiles(cmd string) []string {
	if !goLine.MatchString(cmd) {
		return nil
	}
	// TODO: this does not support quoted files
	nameChunk := goFiles.FindAllStringSubmatch(cmd, -1)
	if len(nameChunk) == 1 && len(nameChunk[0]) == 1 {
		return strings.Split(nameChunk[0][0], " ")
	}
	return nil
}

func (o *Og) GetBuildFiles(steps []*BuildStep) ([]string, error) {
	var names []string
	for _, step := range steps {
		// TODO: this will *not* work on files inside strings?
		for _, cmd := range step.Pure {
			for _, name := range GetCmdFiles(cmd) {
				if !strings.HasPrefix(name, "/") {
					name = path.Join(step.Dir, name)
				}
				names = append(names, name)
			}
		}
	}
	return names, nil
}

func (o *Og) ParseBuild(cmd string, args ...string) ([]*BuildStep, error) {
	args = append([]string{cmd, "-n"}, args...)
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		fmt.Printf("%s", out)
		return nil, err
	}
	scopeRe := regexp.MustCompile(`(?m)^# (_.+|command-line-arguments)$`)
	cdRe := regexp.MustCompile(`(?m)^cd (.+)$`)
	var steps []*BuildStep
	var newStep *BuildStep
	step := &BuildStep{Dir: "."}
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("#")) {
			tmp := scopeRe.FindSubmatch(line)
			if len(tmp) >= 2 {
				newStep = &BuildStep{Scope: string(tmp[1]), Dir: step.Dir}
			}
		} else {
			tmp := cdRe.FindSubmatch(line)
			if len(tmp) >= 2 {
				cd := string(tmp[1])
				if path.Clean(cd) != "." {
					// TODO: make sure this doesn't mess up the directory
					if len(step.Cmds) == 0 || step.Dir == "." {
						step.Dir = cd
					} else {
						newStep = &BuildStep{Scope: step.Scope, Dir: cd}
					}
				}
			}
			if newStep == nil {
				step.Cmds = append(step.Cmds, string(line))
			}
		}
		if newStep != nil {
			if step.Scope != "" && len(step.Cmds) > 0 {
				step.Init()
				steps = append(steps, step)
			}
			step = newStep
			newStep = nil
		}
	}
	if len(step.Cmds) > 0 {
		step.Init()
		steps = append(steps, step)
	}
	return steps, nil
}

func (o *Og) RewriteBuild(steps []*BuildStep) error {
	// TODO: what do ./*.go lines look like with `go build -n` on Windows?
	for _, step := range steps {
		for i, cmd := range step.Cmds {
			if !goLine.Match([]byte(cmd)) {
				continue
			}
			files := GetCmdFiles(cmd)
			for i, name := range files {
				// TODO: instead of escaping, exec commands without a shell
				name = path.Join("_", step.Dir, name)
				files[i] = "\"" + path.Join("$WORK", escapeFilename(name)) + "\""
			}
			suffix := strings.Join(files, " ")
			step.Cmds[i] = goFiles.ReplaceAllString(cmd, "") + suffix
		}
	}
	return nil
}

func (o *Og) RunBuild(steps []*BuildStep) error {
	// TODO: support "don't delete $WORK after build" flag
	noop := false
	for _, v := range o.Args {
		if v == "-n" {
			noop = true
			break
		}
	}
	if noop {
		for _, step := range steps {
			// TODO: make sure this path makes sense on Windows
			fmt.Printf("\n#\n# %s\n#\n\n", step.Scope)
			for _, line := range step.Cmds {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		work, err := ioutil.TempDir("", "")
		if err != nil {
			return err
		}
		names, err := o.GetBuildFiles(steps)
		if err != nil {
			return err
		}
		err = o.GenFiles(work, false, names...)
		if err != nil {
			return err
		}
		env := os.Environ()
		env = append(env, "WORK="+work)
		for _, step := range steps {
			for _, line := range step.Cmds {
				cmd := exec.Command("sh", "-c", line)
				cmd.Env = env
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run();
				if err != nil {
					log.Fatal(err)
				}
				if err != nil && !strings.HasPrefix(line, "mkdir") {
					log.Fatal(err)
				}
			}
		}
		os.RemoveAll(work)
	}
	return nil
}
