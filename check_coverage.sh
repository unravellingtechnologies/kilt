#!/bin/bash

# Script to check comprehensive docstring coverage

echo "Checking comprehensive docstring coverage..."
echo "=================================================="

total_functions=0
functions_with_docs=0

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
    
    # Use awk to process the file and count functions with/without docstrings
    result=$(awk '
    BEGIN { func_count = 0; doc_count = 0; in_comment = 0 }
    
    # Track if we are in a multi-line comment
    /^\/\*/ { in_comment = 1 }
    /\*\// { in_comment = 0; next }
    in_comment == 1 { next }
    
    # Look for function declarations
    /^func / {
        func_count++
        
        # Check if the previous line was a comment (docstring)
        if (prev_line ~ /^\/\//) {
            doc_count++
        }
        
        # Check two lines back in case there is a blank line
        else if (prev_prev_line ~ /^\/\// && prev_line ~ /^[[:space:]]*$/) {
            doc_count++
        }
    }
    
    # Store previous lines
    { prev_prev_line = prev_line; prev_line = $0 }
    
    END { print func_count "," doc_count }
    ' "$file")
    
    func_count=$(echo "$result" | cut -d',' -f1)
    doc_count=$(echo "$result" | cut -d',' -f2)
    
    ((total_functions += func_count))
    ((functions_with_docs += doc_count))
    
    echo "  Functions: $func_count, With docs: $doc_count"
done

echo "=================================================="
echo "Final Summary:"
echo "  Total functions: $total_functions"
echo "  Functions with docstrings: $functions_with_docs"

if [ $total_functions -gt 0 ]; then
    coverage=$(( functions_with_docs * 100 / total_functions ))
    echo "  Docstring coverage: $coverage%"
    
    if [ $coverage -ge 80 ]; then
        echo "  ✅ Meets 80% threshold!"
    else
        echo "  ❌ Below 80% threshold"
    fi
else
    echo "  Docstring coverage: N/A"
fi
