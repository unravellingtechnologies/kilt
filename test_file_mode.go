package main

import (
	"fmt"
	"regexp"
)

func isValidFileMode(mode string) bool {
	// Should be 3-4 digits in octal format (e.g., "0644", "755", "077")
	// 3-digit modes may begin with 0 (e.g., "755", "644", "077")
	// 4-digit modes should start with 0 (e.g., "0644", "0755")
	if len(mode) == 3 {
		// 3 digits, all positions may be 0-7
		matched, _ := regexp.MatchString(`^[0-7]{3}$`, mode)
		return matched
	}
	if len(mode) == 4 {
		// 4 digits, must start with 0
		matched, _ := regexp.MatchString(`^0[0-7]{3}$`, mode)
		return matched
	}
	return false
}

func main() {
	testCases := []struct {
		mode  string
		valid bool
	}{
		// Valid 3-digit modes
		{"755", true},
		{"644", true},
		{"777", true},
		{"000", true},
		{"077", true}, // This was previously invalid
		{"123", true},
		
		// Valid 4-digit modes
		{"0644", true},
		{"0755", true},
		{"0777", true},
		{"0000", true},
		
		// Invalid modes
		{"888", false}, // Invalid octal digit
		{"12", false},   // Too short
		{"12345", false}, // Too long
		{"abc", false},  // Non-numeric
		{"755", true},   // Valid
		{"1755", false}, // 4 digits not starting with 0
	}
	
	fmt.Println("Testing file mode validation:")
	for _, tc := range testCases {
		result := isValidFileMode(tc.mode)
		status := "✓"
		if result != tc.valid {
			status = "✗"
		}
		fmt.Printf("%s %s: expected %v, got %v\n", status, tc.mode, tc.valid, result)
	}
}
