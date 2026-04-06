package prompt

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

// Confirm asks the user a yes/no question. Returns true for yes.
// If skipPrompt is true, returns true without asking.
func Confirm(message string, skipPrompt bool) bool {
	if skipPrompt {
		return true
	}

	var confirmed bool
	err := huh.NewConfirm().
		Title(message).
		Value(&confirmed).
		Run()
	if err != nil {
		return false
	}
	return confirmed
}

// Filter shows a fuzzy filter over items and returns the selected item.
// If stdin is not a terminal, returns the best substring match for query.
func Filter(items []string, placeholder string, query string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return bestMatch(items, query), nil
	}

	var selected string
	opts := make([]huh.Option[string], len(items))
	for i, item := range items {
		opts[i] = huh.NewOption(item, item)
	}

	err := huh.NewSelect[string]().
		Title(placeholder).
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

func bestMatch(items []string, query string) string {
	if query == "" && len(items) > 0 {
		return items[0]
	}
	q := strings.ToLower(query)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), q) {
			return item
		}
	}
	if len(items) > 0 {
		return items[0]
	}
	return ""
}

// Log prints a message to stderr (not captured by quiet mode output piping).
func Log(quiet bool, format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// LogError prints an error to stderr.
func LogError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
