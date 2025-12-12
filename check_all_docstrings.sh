#!/bin/bash

# Script to check for missing docstrings in ALL functions in Go files

echo "Checking for missing docstrings in ALL functions in Go files..."
echo "=================================================="

missing_count=0
total_functions=0

# Function to check if a line is a comment
is_comment() {
    [[ "$1" =~ ^[[:space:]]*// ]]
}

# Function to check if a line is empty
is_empty() {
    [[ -z "${1// }" ]]
}

# Check each Go file (excluding test files)
for file in $(find . -name "*.go" -not -name "*_test.go" | sort); do
    echo "Checking $file..."
    
    # Get line numbers of all functions
    grep -n "^func " "$file" | while IFS=: read -r lineno func_line; do
        ((total_functions++))
        
        # Check if the previous line has a docstring
        prev_lineno=$((lineno - 1))
        
        if [ $prev_lineno -gt 0 ]; then
            prev_line=$(sed -n "${prev_lineno}p" "$file")
            
            # Skip if previous line is a comment (docstring)
            if is_comment "$prev_line"; then
                continue
            fi
            
            # Check one more line back in case there's a blank line
            prev_prev_lineno=$((lineno - 2))
            if [ $prev_prev_lineno -gt 0 ]; then
                prev_prev_line=$(sed -n "${prev_prev_lineno}p" "$file")
                if is_comment "$prev_prev_line" && is_empty "$prev_line"; then
                    continue
                fi
            fi
        fi
        
        # Extract function name for reporting
        func_name=$(echo "$func_line" | sed 's/func //' | sed 's/(.*//')
        echo "  MISSING: $func_name at line $lineno"
        ((missing_count++))
    done
done

echo "=================================================="
echo "Summary:"
echo "  Total functions: $total_functions"
echo "  Missing docstrings: $missing_count"
if [ $total_functions -gt 0 ]; then
    coverage=$(( (total_functions - missing_count) * 100 / total_functions ))
    echo "  Current coverage: $coverage%"
else
    echo "  Current coverage: N/A"
fi
