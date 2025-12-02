// Package logger provides structured logging with colour support for Kilt.
package logger

import (
	"errors"
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

// FormatError formats an error with colour support
func FormatError(err error, colourized bool) string {
	if err == nil {
		return ""
	}

	var colourReset, colourRed, colourBold string
	if colourized {
		colourReset = "\033[0m"
		colourRed = "\033[31m"
		colourBold = "\033[1m"
	}

	msg := err.Error()

	// Check if it's an ErrorWithSuggestion
	ews := &ErrorWithSuggestion{}
	if errors.As(err, &ews) {
		parts := strings.Split(ews.Err.Error(), "\n")
		errorMsg := parts[0]
		rest := strings.Join(parts[1:], "\n")

		formatted := colourRed + colourBold + "Error: " + colourReset + colourRed + errorMsg + colourReset
		if ews.Suggestion != "" {
			formatted += "\n\n" + colourBold + "Suggestion: " + colourReset + ews.Suggestion
		}
		if rest != "" {
			formatted += "\n" + rest
		}
		return formatted
	}

	return colourRed + colourBold + "Error: " + colourReset + colourRed + msg + colourReset
}
