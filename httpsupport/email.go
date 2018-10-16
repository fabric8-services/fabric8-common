package httpsupport

import "regexp"

// ValidateEmail return true if the string is a valid email address
// This is a very simple validation. It just checks if there is @ and dot in the address
func ValidateEmail(email string) (bool, error) {
	// .+@.+\..+
	return regexp.MatchString(".+@.+\\..+", email)
}
