// Package logger provides structured logging with color support for Kilt.
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ProgressBar represents a progress bar
type ProgressBar struct {
	total     int
	current   int
	width     int
	output    io.Writer
	colorized bool
	label     string
	startTime time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, label string, colorized bool) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     50,
		output:    os.Stderr,
		colorized: colorized && isTerminal(os.Stderr),
		label:     label,
		startTime: time.Now(),
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int) {
	p.current = current
	p.render()
}

// Increment increments the progress bar by 1
func (p *ProgressBar) Increment() {
	p.current++
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// SetTotal sets the total value
func (p *ProgressBar) SetTotal(total int) {
	p.total = total
	p.render()
}

// render renders the progress bar
func (p *ProgressBar) render() {
	if p.total == 0 {
		return
	}

	percentage := float64(p.current) / float64(p.total)
	filled := int(percentage * float64(p.width))
	empty := p.width - filled

	var bar strings.Builder

	// Label
	if p.label != "" {
		bar.WriteString(p.label + " ")
	}

	// Progress bar
	bar.WriteString("[")
	if p.colorized {
		bar.WriteString("\033[32m") // Green
	}
	bar.WriteString(strings.Repeat("=", filled))
	if p.colorized {
		bar.WriteString("\033[0m") // Reset
	}
	bar.WriteString(strings.Repeat(" ", empty))
	bar.WriteString("]")

	// Percentage and count
	bar.WriteString(fmt.Sprintf(" %3.0f%% (%d/%d)", percentage*100, p.current, p.total))

	// Elapsed time
	elapsed := time.Since(p.startTime)
	if p.current > 0 && p.current < p.total {
		estimatedTotal := time.Duration(float64(elapsed) / percentage)
		remaining := estimatedTotal - elapsed
		bar.WriteString(fmt.Sprintf(" [%s remaining]", formatDuration(remaining)))
	} else if p.current == p.total {
		bar.WriteString(fmt.Sprintf(" [%s]", formatDuration(elapsed)))
	}

	// Move cursor to beginning of line and clear
	fmt.Fprintf(p.output, "\r\033[K%s", bar.String())
}

// Finish finishes the progress bar
func (p *ProgressBar) Finish() {
	p.current = p.total
	p.render()
	fmt.Fprintln(p.output) // New line
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Spinner represents a spinner for indeterminate progress
type Spinner struct {
	output    io.Writer
	colorized bool
	message   string
	stop      chan bool
	done      chan bool
	frames    []string
	frame     int
}

// NewSpinner creates a new spinner
func NewSpinner(message string, colorized bool) *Spinner {
	return &Spinner{
		output:    os.Stderr,
		colorized: colorized && isTerminal(os.Stderr),
		message:   message,
		stop:      make(chan bool),
		done:      make(chan bool),
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		frame:     0,
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	go s.run()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.stop <- true
	<-s.done
	fmt.Fprintf(s.output, "\r\033[K") // Clear line
}

// run runs the spinner animation
func (s *Spinner) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			s.done <- true
			return
		case <-ticker.C:
			frame := s.frames[s.frame%len(s.frames)]
			if s.colorized {
				frame = "\033[36m" + frame + "\033[0m" // Cyan
			}
			fmt.Fprintf(s.output, "\r\033[K%s %s", frame, s.message)
			s.frame++
		}
	}
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.message = message
}

