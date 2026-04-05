package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RequiresLibraryRoots(t *testing.T) {
	t.Setenv("MYLIB_LIBRARY_ROOTS", "")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MYLIB_LIBRARY_ROOTS")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("MYLIB_LIBRARY_ROOTS", t.TempDir())
	t.Setenv("MYLIB_DATA_DIR", "")
	t.Setenv("MYLIB_LISTEN", "")
	t.Setenv("MYLIB_SCAN_INTERVAL", "")
	t.Setenv("MYLIB_LOG_LEVEL", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.LibraryRoots, 1)
	assert.Equal(t, ":8080", cfg.Listen)
	assert.Equal(t, 10*time.Minute, cfg.ScanInterval)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_MultipleRoots(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	t.Setenv("MYLIB_LIBRARY_ROOTS", a+":"+b)
	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.LibraryRoots, 2)
}
