package container_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suniastar/testsetup/container"
)

func TestAutoGuessHostname(t *testing.T) {
	require.NoError(t, os.Setenv("GITLAB_CI", ""))
	assert.Equal(t, "localhost", container.AutoGuessHostname())

	require.NoError(t, os.Setenv("GITLAB_CI", "true"))
	assert.Equal(t, "docker", container.AutoGuessHostname())
}
