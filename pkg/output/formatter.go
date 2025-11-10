package output

import (
	"encoding/json"
	"fmt"
)

// Color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Success prints a success message in green
func Success(message string) {
	fmt.Printf("%s%s%s\n", ColorGreen, message, ColorReset)
}

// Error prints an error message in red
func Error(message string) {
	fmt.Printf("%s%s%s\n", ColorRed, message, ColorReset)
}

// Info prints an info message in cyan
func Info(message string) {
	fmt.Printf("%s%s%s\n", ColorCyan, message, ColorReset)
}

// Warning prints a warning message in yellow
func Warning(message string) {
	fmt.Printf("%s%s%s\n", ColorYellow, message, ColorReset)
}

// Header prints a section header in blue
func Header(message string) {
	fmt.Printf("\n%s═══ %s ═══%s\n\n", ColorBlue, message, ColorReset)
}

// JSON prints formatted JSON
func JSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// Table prints a simple key-value table
func Table(rows map[string]string) {
	for key, value := range rows {
		fmt.Printf("%s%-20s%s: %s\n", ColorGray, key, ColorReset, value)
	}
}
