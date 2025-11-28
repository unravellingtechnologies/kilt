// Package logger provides structured logging with color support for Kilt.
package logger

import (
	"fmt"
	"strings"
)

// ErrorWithSuggestion wraps an error with a helpful suggestion
type ErrorWithSuggestion struct {
	Err        error
	Suggestion string
	Context    map[string]interface{}
}

// Error returns the error message
func (e *ErrorWithSuggestion) Error() string {
	msg := e.Err.Error()
	if e.Suggestion != "" {
		msg += "\n\nSuggestion: " + e.Suggestion
	}
	if len(e.Context) > 0 {
		var ctxParts []string
		for k, v := range e.Context {
			ctxParts = append(ctxParts, fmt.Sprintf("%s=%v", k, v))
		}
		msg += "\nContext: " + strings.Join(ctxParts, ", ")
	}
	return msg
}

// Unwrap returns the underlying error
func (e *ErrorWithSuggestion) Unwrap() error {
	return e.Err
}

// WithSuggestion wraps an error with a suggestion
func WithSuggestion(err error, suggestion string) error {
	if err == nil {
		return nil
	}
	return &ErrorWithSuggestion{
		Err:        err,
		Suggestion: suggestion,
	}
}

// WithSuggestionAndContext wraps an error with a suggestion and context
func WithSuggestionAndContext(err error, suggestion string, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return &ErrorWithSuggestion{
		Err:        err,
		Suggestion: suggestion,
		Context:    context,
	}
}

// FormatError formats an error with color support
func FormatError(err error, colorized bool) string {
	if err == nil {
		return ""
	}

	var colorReset, colorRed, colorBold string
	if colorized {
		colorReset = "\033[0m"
		colorRed = "\033[31m"
		colorBold = "\033[1m"
	}

	msg := err.Error()

	// Check if it's an ErrorWithSuggestion
	if ews, ok := err.(*ErrorWithSuggestion); ok {
		parts := strings.Split(ews.Err.Error(), "\n")
		errorMsg := parts[0]
		rest := strings.Join(parts[1:], "\n")

		formatted := colorRed + colorBold + "Error: " + colorReset + colorRed + errorMsg + colorReset
		if ews.Suggestion != "" {
			formatted += "\n\n" + colorBold + "Suggestion: " + colorReset + ews.Suggestion
		}
		if rest != "" {
			formatted += "\n" + rest
		}
		return formatted
	}

	return colorRed + colorBold + "Error: " + colorReset + colorRed + msg + colorReset
}


