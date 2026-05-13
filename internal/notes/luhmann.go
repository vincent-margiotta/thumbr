package notes

import (
	"strconv"
	"unicode"
)

// DeriveContinuation returns the Luhmann continuation address for stem.
// A digit-ending stem gets "a" appended; a letter-ending stem gets "1" appended.
// Returns ("", false) if the stem is empty or contains no trailing digit/letter run.
func DeriveContinuation(stem string) (string, bool) {
	prefix, last, isDigit, ok := SplitLuhmannStem(stem)
	if !ok {
		return "", false
	}
	if isDigit {
		return prefix + last + "a", true
	}
	return prefix + last + "1", true
}

// DeriveBranch returns the Luhmann sibling address for stem.
// A digit-ending stem has its trailing number incremented; a single-letter-ending stem
// has its letter incremented. Returns ("", false) when the pattern doesn't match or
// the letter would overflow past 'z'/'Z'.
func DeriveBranch(stem string) (string, bool) {
	prefix, last, isDigit, ok := SplitLuhmannStem(stem)
	if !ok {
		return "", false
	}
	if isDigit {
		n, err := strconv.Atoi(last)
		if err != nil {
			return "", false
		}
		return prefix + strconv.Itoa(n+1), true
	}
	// Letter component: only single letters are supported for branching.
	if len(last) != 1 {
		return "", false
	}
	r := rune(last[0])
	next := r + 1
	if unicode.IsLower(r) && next > 'z' {
		return "", false
	}
	if unicode.IsUpper(r) && next > 'Z' {
		return "", false
	}
	return prefix + string(next), true
}

// SplitLuhmannStem splits stem into prefix and a trailing run of uniform type.
// The trailing run is either all-digits or all-letters (not mixed).
// Returns (prefix, lastRun, isDigit, ok); ok=false means no valid trailing run.
// The entire stem must be alphanumeric — any special character causes ok=false.
func SplitLuhmannStem(stem string) (prefix, last string, isDigit bool, ok bool) {
	if stem == "" {
		return "", "", false, false
	}
	for _, r := range stem {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", "", false, false
		}
	}
	lastRune := rune(stem[len(stem)-1])
	trailingDigit := unicode.IsDigit(lastRune)
	trailingLetter := unicode.IsLetter(lastRune)
	if !trailingDigit && !trailingLetter {
		return "", "", false, false
	}
	i := len(stem)
	for i > 0 {
		r := rune(stem[i-1])
		if trailingDigit && !unicode.IsDigit(r) {
			break
		}
		if trailingLetter && !unicode.IsLetter(r) {
			break
		}
		i--
	}
	return stem[:i], stem[i:], trailingDigit, true
}
