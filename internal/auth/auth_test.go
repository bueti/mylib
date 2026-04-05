package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := HashPassword("s3cret!")
	require.NoError(t, err)
	assert.NotEmpty(t, h)
	assert.NoError(t, VerifyPassword(h, "s3cret!"))
	assert.ErrorIs(t, VerifyPassword(h, "wrong"), ErrBadPassword)
}

func TestHashPassword_Empty(t *testing.T) {
	_, err := HashPassword("")
	require.Error(t, err)
}

func TestNewSessionToken_Unique(t *testing.T) {
	a, err := NewSessionToken()
	require.NoError(t, err)
	b, err := NewSessionToken()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.Len(t, a, 64) // 32 bytes hex
}
