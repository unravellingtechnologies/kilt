package main

import (
	"fmt"
	"strings"
)

func main() {
	// Test the new parsing logic
	testCases := []string{
		"M  simple.txt",
		"A  file with spaces.txt", 
		"R  old name.txt -> new name.txt",
		"?? untracked file with spaces.txt",
		"MM modified.txt",
	}

	for _, line := range testCases {
		if strings.TrimSpace(line) != "" && len(line) >= 3 {
			// Extract filename from porcelain format (skip 2-char status prefix + space)
			filename := strings.TrimSpace(line[3:])

			// Handle renames: extract target name after " -> "
			if strings.Contains(filename, " -> ") {
				parts := strings.Split(filename, " -> ")
				if len(parts) >= 2 {
					filename = strings.TrimSpace(parts[len(parts)-1])
				}
			}

			fmt.Printf("Input: %q -> Output: %q\n", line, filename)
		}
	}
}
