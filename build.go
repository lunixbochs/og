package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type BuildStep struct {
	Dir  string
	Cmds []string
	Pure []string
}

func NewBuildStep(dir string, cmds []string) *BuildStep {
	pure := make([]string, len(cmds))
	copy(pure, cmds)
	return &BuildStep{dir, cmds, pure}
}

func (o *Og) GetBuildFiles(steps []*BuildStep) ([]string, error) {
	var names []string
	goLine := regexp.MustCompile(`(?m)^.+?(\./.+?\.go).*?$`)
	goFiles := regexp.MustCompile(`(?m)\./.+$`)
	for _, step := range steps {
		// TODO: this will *not* work on files inside strings?
		for _, cmd := range step.Pure {
			if !goLine.MatchString(cmd) {
				continue
			}
			// TODO: this does not support quoted files
			nameChunk := goFiles.FindAllStringSubmatch(cmd, -1)
			if len(nameChunk) == 1 {
				names = append(names, strings.Split(nameChunk[0][0], " ")...)
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
	prefixRe := regexp.MustCompile(`(?m)^# _(.+)$`)
	var steps []*BuildStep
	var prefix string
	var cur []string
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("#")) {
			tmp := prefixRe.FindSubmatch(line)
			if len(tmp) > 0 {
				if prefix != "" && len(cur) > 0 {
					steps = append(steps, NewBuildStep(prefix, cur))
					cur = nil
				}
				prefix = string(tmp[1])
			}
		} else {
			cur = append(cur, string(line))
		}
	}
	if len(cur) > 0 {
		steps = append(steps, NewBuildStep(prefix, cur))
	}
	return steps, nil
}

func (o *Og) RewriteBuild(steps []*BuildStep) error {
	goLine := regexp.MustCompile(`(?m)^.+?(\./.+?\.go).*?$`)
	goNuke := regexp.MustCompile(`(?m)\./.+$`)
	// TODO: what do ./*.go lines look like with `go build -n` on Windows?
	for _, step := range steps {
		prefix := step.Dir
		for i, cmd := range step.Cmds {
			if !goLine.Match([]byte(cmd)) {
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
			suffix := strings.Join(files, " ")
			step.Cmds[i] = goNuke.ReplaceAllString(cmd, "") + suffix
		}
	}
	return nil
}

func (o *Og) RunBuild(steps []*BuildStep) error {
	// TODO: support "don't delete $WORK after build" flag
	var noop bool
	flagset := flag.NewFlagSet("build", flag.ExitOnError)
	flagset.BoolVar(&noop, "n", false, "")
	flagset.Parse(o.Args)
	if noop {
		for _, step := range steps {
			// TODO: make sure this path makes sense on Windows
			fmt.Printf("\n#\n# _%s\n#\n\n", step.Dir)
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
		err = o.GenFiles(work, names...)
		if err != nil {
			return err
		}
		env := os.Environ()
		env = append(env, "WORK="+work)
		for _, step := range steps {
			for _, line := range step.Cmds {
				cmd := exec.Command("sh", "-c", line)
				cmd.Env = env
				out, err := cmd.CombinedOutput()
				out = bytes.TrimSpace(out)
				if len(out) > 0 {
					fmt.Printf("%s\n", out)
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
