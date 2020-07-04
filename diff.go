package main

import (
	"bytes"
	"fmt"
	diffPkg "github.com/j-keck/zfs-snap-diff/pkg/diff"
	"strings"
)

func diffsPrettyText(diff diffPkg.Diff, colored bool) string {

	var buff bytes.Buffer
	for n, deltas := range diff.Deltas {
		header := fmt.Sprintf("Chunk %d - starting at line %d", n, deltas[0].LineNrFrom)
		buff.WriteString(strings.Repeat("=", len(header)) + "\n")
		buff.WriteString(header + "\n")
		buff.WriteString(strings.Repeat("-", len(header)) + "\n")
		buff.WriteString(diffPrettyText(deltas, colored))
		buff.WriteString("\n")
	}

	return buff.String()
}

func diffPrettyText(deltas diffPkg.Deltas, colored bool) string {
	var buff bytes.Buffer

	for _, delta := range deltas {
		switch delta.Type {
		case diffPkg.Ins:
			if colored {
				buff.WriteString("\x1b[32m")
				buff.WriteString(delta.Text)
				buff.WriteString("\x1b[0m")
			} else {
				for n, line := range strings.Split(delta.Text, "\n") {
					if n > 0 {
						buff.WriteString("\n")
					}
					if len(line) > 0 {
						buff.WriteString("+ " + line)
					}
				}
			}
		case diffPkg.Del:
			if colored {
				buff.WriteString("\x1b[31m")
				buff.WriteString(delta.Text)
				buff.WriteString("\x1b[0m")
			} else {
				for n, line := range strings.Split(delta.Text, "\n") {
					if n > 0 {
						buff.WriteString("\n")
					}
					if len(line) > 0 {
						buff.WriteString("- " + line)
					}
				}
			}
		case diffPkg.Eq:
			if colored {
				buff.WriteString(delta.Text)
			} else {
				buff.WriteString("  " + delta.Text)
			}
		}
	}
	return buff.String()
}
