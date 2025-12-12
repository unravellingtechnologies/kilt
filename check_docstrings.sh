#!/bin/bash

# Script to check for missing docstrings in Go files

echo "Checking for missing docstrings in Go files..."
echo "=================================================="

missing_count=0
total_exported=0

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
    
    # Get line numbers of exported functions/methods
    # Pattern 1: func FunctionName
    # Pattern 2: func (receiver) MethodName
    grep -n "^func [A-Z]\|^func (.*) [A-Z]" "$file" | while IFS=: read -r lineno func_line; do
        ((total_exported++))
        
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
        if [[ "$func_line" =~ "func (.*) ([A-Z][^("]*" ]]; then
            # Method: func (receiver) MethodName
            func_name="${BASH_REMATCH[2]}"
            receiver="${BASH_REMATCH[1]}"
            echo "  MISSING: ${receiver}.${func_name} at line $lineno"
        else
            # Function: func FunctionName
            func_name=$(echo "$func_line" | sed 's/func //' | sed 's/(.*//')
            echo "  MISSING: $func_name at line $lineno"
        fi
        
        ((missing_count++))
    done
done

echo "=================================================="
echo "Summary:"
echo "  Total exported functions/methods: $total_exported"
echo "  Missing docstrings: $missing_count"
if [ $total_exported -gt 0 ]; then
    coverage=$(( (total_exported - missing_count) * 100 / total_exported ))
    echo "  Current coverage: $coverage%"
else
    echo "  Current coverage: N/A"
fi
