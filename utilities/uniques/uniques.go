// uniques contains global constants that must be referenced across multiple packages
package uniques

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	// the string format the Gravwell client requires
	SearchTimeFormat = "2006-01-02T15:04:05.999999999Z07:00"
)

// Returns a string about ignoring a flag due to causeFlag's existance
func WarnFlagIgnore(ignoredFlag, causeFlag string) string {
	return fmt.Sprintf("WARN: ignoring flag --%v due to --%v", ignoredFlag, causeFlag)
}

// Validator function for a TI intended to consume cron-like input.
// For efficiencies sake, it only evaluates the end rune.
// Checking the values of each complete word is delayed until connection.CreateScheduledSearch to
// save on cycles.
func CronRuneValidator(s string) error {
	// check for an empty TI
	if strings.TrimSpace(s) == "" {
		return nil
	}
	runes := []rune(s)
	if len(runes) < 1 {
		return nil
	}

	// check that the latest input is a digit or space
	if char := runes[len(runes)-1]; !unicode.IsSpace(char) &&
		!unicode.IsDigit(rune(char)) && char != '*' {
		return errors.New("frequency can contain only digits or '*'")
	}

	// check that we do not have too many values
	exploded := strings.Split(s, " ")
	if len(exploded) > 5 {
		return errors.New("must be exactly 5 values")
	}

	// check that the newest word is <= 2 characters
	lastWord := []rune(exploded[len(exploded)-1])
	if len(lastWord) > 2 {
		return errors.New("each word is <= 2 digits")
	}

	return nil
}
