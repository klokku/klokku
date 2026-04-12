package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{
		Server: "https://app.klokku.com",
		Token:  "my-token",
	}
	err := Save(path, cfg)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}

func TestLoad_FileNotExist(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, Config{}, cfg)
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	err := Save(path, Config{Server: "http://localhost"})
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestResolve_Priority(t *testing.T) {
	fileCfg := Config{
		Server: "file-server",
		Token:  "file-token",
	}

	// Env overrides file
	t.Setenv("KLOKKU_SERVER", "env-server")
	t.Setenv("KLOKKU_TOKEN", "env-token")

	resolved := Resolve(fileCfg, "", "", "")
	assert.Equal(t, "env-server", resolved.Server)
	assert.Equal(t, "env-token", resolved.Token)

	// Flags override env
	resolved = Resolve(fileCfg, "flag-server", "", "flag-token")
	assert.Equal(t, "flag-server", resolved.Server)
	assert.Equal(t, "flag-token", resolved.Token)
}

func TestResolve_EnvUserID(t *testing.T) {
	t.Setenv("KLOKKU_USER_ID", "env-uid")
	resolved := Resolve(Config{}, "", "", "")
	assert.Equal(t, "env-uid", resolved.UserID)
}

func TestValidate_NoServer(t *testing.T) {
	err := Validate(Config{Token: "tok"})
	assert.ErrorContains(t, err, "server URL is required")
}

func TestValidate_BothAuthMethods(t *testing.T) {
	err := Validate(Config{Server: "http://x", Token: "tok", UserID: "uid"})
	assert.ErrorContains(t, err, "cannot set both")
}

func TestValidate_NoAuth(t *testing.T) {
	err := Validate(Config{Server: "http://x"})
	assert.ErrorContains(t, err, "authentication required")
}

func TestValidate_Token(t *testing.T) {
	err := Validate(Config{Server: "http://x", Token: "tok"})
	assert.NoError(t, err)
}

func TestValidate_UserID(t *testing.T) {
	err := Validate(Config{Server: "http://x", UserID: "uid"})
	assert.NoError(t, err)
}
