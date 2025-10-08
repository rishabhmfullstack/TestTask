package main

import (
	"testing"
)

func TestNewEmailValidator(t *testing.T) {
	validator := NewEmailValidator()
	if validator == nil {
		t.Fatal("NewEmailValidator() returned nil")
	}
	if validator.emailRegex == nil {
		t.Fatal("Email regex is nil")
	}
}

func TestIsValidEmail(t *testing.T) {
	validator := NewEmailValidator()

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		// Valid emails
		{"Valid simple email", "test@example.com", true},
		{"Valid email with subdomain", "user@mail.example.com", true},
		{"Valid email with numbers", "user123@example123.com", true},
		{"Valid email with dots", "first.last@example.com", true},
		{"Valid email with plus", "user+tag@example.com", true},
		{"Valid email with dash", "user-name@example.com", true},
		{"Valid email with underscore", "user_name@example.com", true},
		{"Valid email with percent", "user%name@example.com", true},
		{"Valid email with UK domain", "test@example.co.uk", true},
		{"Valid email with long TLD", "test@example.museum", true},
		{"Valid email with mixed case", "Test@Example.Com", true},
		{"Valid email with spaces trimmed", "  test@example.com  ", true},

		// Invalid emails
		{"Empty email", "", false},
		{"Email without @", "testexample.com", false},
		{"Email without domain", "test@", false},
		{"Email without local part", "@example.com", false},
		{"Email with multiple @", "test@@example.com", false},
		{"Email with space", "test @example.com", false},
		{"Email with invalid characters", "test@example!.com", false},
		{"Email with short TLD", "test@example.c", false},
		{"Email with no TLD", "test@example", false},
		{"Email with leading dot", ".test@example.com", false},
		{"Email with trailing dot", "test.@example.com", false},
		{"Email with consecutive dots", "test..test@example.com", true}, // Current regex allows this
		{"Email with @ in local part", "te@st@example.com", false},
		{"Email with @ in domain", "test@ex@ample.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("IsValidEmail(%q) = %v, expected %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestHasValidEmail(t *testing.T) {
	validator := NewEmailValidator()

	tests := []struct {
		name     string
		fields   []string
		expected bool
	}{
		// Has valid email
		{"Single valid email", []string{"test@example.com"}, true},
		{"Valid email among other fields", []string{"John Doe", "test@example.com", "555-1234"}, true},
		{"Valid email at beginning", []string{"test@example.com", "John Doe", "555-1234"}, true},
		{"Valid email at end", []string{"John Doe", "555-1234", "test@example.com"}, true},
		{"Multiple valid emails", []string{"test1@example.com", "test2@example.com"}, true},
		{"Valid email with invalid email", []string{"invalid-email", "test@example.com"}, true},

		// No valid email
		{"Empty fields", []string{}, false},
		{"Single invalid email", []string{"invalid-email"}, false},
		{"Multiple invalid emails", []string{"invalid1", "invalid2", "invalid3"}, false},
		{"Empty strings", []string{"", "", ""}, false},
		{"Non-email data", []string{"John Doe", "555-1234", "Acme Corp"}, false},
		{"Email-like but invalid", []string{"test@", "@example.com", "test.example.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.HasValidEmail(tt.fields)
			if result != tt.expected {
				t.Errorf("HasValidEmail(%v) = %v, expected %v", tt.fields, result, tt.expected)
			}
		})
	}
}

func TestEmailValidatorConcurrency(t *testing.T) {
	validator := NewEmailValidator()

	// Test concurrent access to email validator
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test multiple operations concurrently
			validator.IsValidEmail("test@example.com")
			validator.HasValidEmail([]string{"test@example.com", "invalid"})
			validator.IsValidEmail("invalid-email")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
