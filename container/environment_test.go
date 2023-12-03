package container_test

import (
	"os"
	"testing"

	"github.com/4ND3R50N/testsetup/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoGuessHostname(t *testing.T) {
	require.NoError(t, os.Setenv("GITLAB_CI", ""))
	assert.Equal(t, "localhost", container.AutoGuessHostname())

	require.NoError(t, os.Setenv("GITLAB_CI", "true"))
	assert.Equal(t, "docker", container.AutoGuessHostname())
}
