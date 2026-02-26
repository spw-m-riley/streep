package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const updateUsage = `Check for a newer version of streep.

Usage:
  streep update
`

// httpClient is used for the GitHub API call; replaceable in tests.
var httpClient interface {
	Get(url string) (*http.Response, error)
} = http.DefaultClient

func executeUpdate(_ []string, stdout io.Writer, _ io.Writer) error {
	if Version == "dev" {
		fmt.Fprintln(stdout, "Running a dev build — skipping update check.")
		return nil
	}

	resp, err := httpClient.Get("https://api.github.com/repos/spw-m-riley/streep/releases/latest")
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	latest := release.TagName
	current := "v" + Version
	if latest == current {
		fmt.Fprintf(stdout, "streep %s is up to date.\n", Version)
		return nil
	}

	fmt.Fprintf(stdout, "A new version of streep is available: %s (you have %s)\n", latest, Version)
	fmt.Fprintf(stdout, "Upgrade:\n")
	fmt.Fprintf(stdout, "  Homebrew:   brew upgrade streep\n")
	fmt.Fprintf(stdout, "  go install: go install github.com/spw-m-riley/streep@latest\n")
	fmt.Fprintf(stdout, "Release notes: %s\n", release.HTMLURL)
	return nil
}
