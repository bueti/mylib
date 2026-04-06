package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizer_AdminPermissions(t *testing.T) {
	az, err := New()
	require.NoError(t, err)

	assert.True(t, az.Can("admin", "books", "read"))
	assert.True(t, az.Can("admin", "books", "delete"))
	assert.True(t, az.Can("admin", "users", "manage"))
	assert.True(t, az.Can("admin", "admin", "access"))
	assert.True(t, az.Can("admin", "scan", "trigger"))
}

func TestAuthorizer_ReaderPermissions(t *testing.T) {
	az, err := New()
	require.NoError(t, err)

	assert.True(t, az.Can("reader", "books", "read"))
	assert.True(t, az.Can("reader", "books", "edit"))
	assert.True(t, az.Can("reader", "books", "upload"))
	assert.True(t, az.Can("reader", "books", "enrich"))
	assert.True(t, az.Can("reader", "scan", "trigger"))

	// Reader cannot delete or manage users.
	assert.False(t, az.Can("reader", "books", "delete"))
	assert.False(t, az.Can("reader", "users", "manage"))
	assert.False(t, az.Can("reader", "admin", "access"))
}

func TestAuthorizer_UnknownRole(t *testing.T) {
	az, err := New()
	require.NoError(t, err)

	assert.False(t, az.Can("unknown", "books", "read"))
}

func TestAuthorizer_PermissionsForRole(t *testing.T) {
	az, err := New()
	require.NoError(t, err)

	readerPerms := az.PermissionsForRole("reader")
	assert.Contains(t, readerPerms, "books:read")
	assert.Contains(t, readerPerms, "books:edit")
	assert.NotContains(t, readerPerms, "books:delete")
	assert.NotContains(t, readerPerms, "users:manage")

	adminPerms := az.PermissionsForRole("admin")
	assert.Contains(t, adminPerms, "books:delete")
	assert.Contains(t, adminPerms, "users:manage")
}
