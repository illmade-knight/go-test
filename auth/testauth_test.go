package auth_test

import (
	"errors"
	"testing"

	"github.com/illmade-knight/go-test/auth"
	"github.com/stretchr/testify/assert"
)

func TestFormatGCPAuthError(t *testing.T) {
	// --- Arrange ---
	// Create a simulated error that mimics what the Google Cloud client library
	// returns when Application Default Credentials are not found.
	originalError := errors.New("google: could not find default credentials")

	// --- Act ---
	// Call the now-exported formatting function from the 'auth' package.
	formattedMessage := auth.FormatGCPAuthError(originalError)

	// --- Assert ---
	// Verify that the formatted message contains all the key user-friendly elements.
	assert.Contains(t, formattedMessage, "GCP AUTHENTICATION FAILED!", "Should contain the main header")
	assert.Contains(t, formattedMessage, "To fix this, please run:", "Should contain the call to action")
	assert.Contains(t, formattedMessage, "gcloud auth application-default login", "Should contain the exact command to run")
	assert.Contains(t, formattedMessage, "Original Error: google: could not find default credentials", "Should include the original error for debugging")
}
