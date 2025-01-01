package puff

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// readDescription reads comments based on the file and line number of
// the caller that called the GET, POST, etc. methods on the router. It
// will read upwards of the method call.
func readDescription(file string, lineNumber int) string {
	srcfile, err := os.ReadFile(file)
	if err != nil {
		slog.Error(fmt.Sprintf("puff: readDescription cannot read description: os.ReadFile failed with error: %s", err.Error()))
		return ""
	}
	lines := strings.Split(string(srcfile), "\n")
	comments := []string{}
	// read file in reverse
	for i := lineNumber - 2; i >= 0; i-- { // guess and check got us to line number - 2
		line := strings.TrimSpace(lines[i])
		if line == "" { // empty line (ignore)
			continue
		}
		if strings.HasPrefix(line, "//") { // we have reached comment
			comment := strings.TrimSpace(line[2:])
			comments = append([]string{comment}, comments...) // since we're reading in reverse, we have to prepend
			continue
		}
		break // since it wasn't an empty line or a comment, we have reached actual code. we are no longer in a comment
	}
	return strings.Join(comments, " ") // trim space removed the spaces
}
