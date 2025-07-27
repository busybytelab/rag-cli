package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	// Colors for different output types
	InfoColor    = color.New(color.FgBlue)
	SuccessColor = color.New(color.FgGreen)
	WarningColor = color.New(color.FgYellow)
	ErrorColor   = color.New(color.FgRed)
	BoldColor    = color.New(color.Bold)

	// Colors for key-value pairs
	KeyColor   = color.New(color.FgCyan)
	ValueColor = color.New(color.FgWhite)

	// Disable colors flag
	colorsDisabled bool
)

// DisableColors disables color output
func DisableColors() {
	colorsDisabled = true
	color.NoColor = true
}

// EnableColors enables color output
func EnableColors() {
	colorsDisabled = false
	color.NoColor = false
}

// Info prints an info message
func Info(format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Printf(format+"\n", args...)
	} else {
		InfoColor.Printf(format+"\n", args...)
	}
}

// Success prints a success message
func Success(format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Printf(format+"\n", args...)
	} else {
		SuccessColor.Printf(format+"\n", args...)
	}
}

// Warning prints a warning message
func Warning(format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Printf(format+"\n", args...)
	} else {
		WarningColor.Printf(format+"\n", args...)
	}
}

// Error prints an error message
func Error(format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	} else {
		ErrorColor.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Bold prints a bold message
func Bold(format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Printf(format+"\n", args...)
	} else {
		BoldColor.Printf(format+"\n", args...)
	}
}

// Printf prints a formatted message without color
func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println prints a message with newline
func Println(args ...interface{}) {
	fmt.Println(args...)
}

// Fprintf prints to a specific writer
func Fprintf(w *os.File, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...)
}

// Fprintln prints to a specific writer with newline
func Fprintln(w *os.File, args ...interface{}) {
	fmt.Fprintln(w, args...)
}

// Print prints a message without newline
func Print(args ...interface{}) {
	fmt.Print(args...)
}

// KeyValue prints a key-value pair with different colors
func KeyValue(key, value string) {
	if colorsDisabled {
		fmt.Printf("%s: %s\n", key, value)
	} else {
		KeyColor.Printf("%s: ", key)
		ValueColor.Printf("%s\n", value)
	}
}

// KeyValuef prints a formatted key-value pair with different colors
func KeyValuef(key, format string, args ...interface{}) {
	if colorsDisabled {
		fmt.Printf("%s: %s\n", key, fmt.Sprintf(format, args...))
	} else {
		KeyColor.Printf("%s: ", key)
		ValueColor.Printf(format+"\n", args...)
	}
}
