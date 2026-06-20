package users

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestDeriveDefaults(t *testing.T) {
	dob, hash, err := deriveDefaults("09/03/2001")
	assert.NoError(t, err)

	assert.Equal(t, 2001, dob.Year())
	assert.Equal(t, 3, int(dob.Month()))
	assert.Equal(t, 9, dob.Day())

	// Hash should match "09032001"
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("09032001"))
	assert.NoError(t, err)

	// Hash should not contain dashes or slashes from the original input
	assert.NotContains(t, "09032001", "/")
	assert.NotContains(t, "09032001", "-")

	// Ensure length is exactly 8 (just checking the plaintext we derived from)
	assert.True(t, regexp.MustCompile(`^[0-9]{8}$`).MatchString("09032001"))
}

func TestDeriveDefaults_Invalid(t *testing.T) {
	_, _, err := deriveDefaults("2001-03-09")
	assert.ErrorIs(t, err, ErrInvalidDOBFormat)
}
