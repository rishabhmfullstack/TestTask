package main

import (
	"regexp"
	"strings"
)

// EmailValidator handles email validation logic
type EmailValidator struct {
	emailRegex *regexp.Regexp
}

// NewEmailValidator creates a new email validator
func NewEmailValidator() *EmailValidator {
	// More strict email regex pattern that allows + and % in local part
	emailPattern := `^[a-zA-Z0-9]([a-zA-Z0-9._%+-]*[a-zA-Z0-9])?@[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`
	emailRegex := regexp.MustCompile(emailPattern)

	return &EmailValidator{
		emailRegex: emailRegex,
	}
}

// IsValidEmail checks if a string is a valid email address
func (ev *EmailValidator) IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	return ev.emailRegex.MatchString(email)
}

// HasValidEmail checks if any field in a row contains a valid email
func (ev *EmailValidator) HasValidEmail(fields []string) bool {
	for _, field := range fields {
		if ev.IsValidEmail(field) {
			return true
		}
	}
	return false
}
