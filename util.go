package main

import (
	"strings"
)

func escapeFilename(name string) string {
	if strings.ContainsAny(name, "\\\"$") {
		escaper := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "$", "\\$")
		return escaper.Replace(name)
	}
	return name
}
