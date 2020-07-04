package main

import (
	pkgDiff "github.com/j-keck/zfs-snap-diff/pkg/diff"
	"bytes"
	"fmt"
	"strings"
)


func diffPrettyText(diff pkgDiff.Diff, colored bool) string {

	var buff bytes.Buffer
	for n, deltas := range diff.Deltas {
		header := fmt.Sprintf("Chunk %d - starting at line %d", n, deltas[0].LineNrFrom)
		buff.WriteString(strings.Repeat("=", len(header)) + "\n")
		buff.WriteString(header + "\n")
		buff.WriteString(strings.Repeat("-", len(header)) + "\n")
		for _, delta := range deltas {
			switch delta.Type {
			case pkgDiff.Ins:
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
			case pkgDiff.Del:
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
			case pkgDiff.Eq:
				if colored {
					buff.WriteString(delta.Text)
				} else {
					buff.WriteString("  " + delta.Text)
				}
			}
		}
		buff.WriteString("\n")
	}


	return buff.String()
}
