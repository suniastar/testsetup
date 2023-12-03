package container

import "os"

func validateHost(host string) string {
	// Allow manual "overwrite".
	if host != "" {
		return host
	}

	return AutoGuessHostname()
}

// AutoGuessHostname will try to guess the correct hostname where containers are reachable.
// If a CI environment is detected "docker" hostname is assumed (DinD), otherwise "localhost".
func AutoGuessHostname() string {
	// In gitlab CI we use DinD, therefore hostname is "docker".
	if os.Getenv("GITLAB_CI") != "" {
		return "docker"
	}

	// Otherwise assume we are running locally.
	return "localhost"
}
