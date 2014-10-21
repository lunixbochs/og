package main

import (
	"log"
	"os/exec"
	"strings"
	"syscall"
)

func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		} else {
			log.Fatal("syscall.WaitStatus doesn't seem to be supported on your platform")
		}
	}
	return 0
}

func escapeFilename(name string) string {
	if strings.ContainsAny(name, "\\\"$") {
		escaper := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "$", "\\$")
		return escaper.Replace(name)
	}
	return name
}
