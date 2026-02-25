package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	tagName string
	htmlURL string
}

func (m *mockHTTPClient) Get(_ string) (*http.Response, error) {
	payload, _ := json.Marshal(map[string]string{
		"tag_name": m.tagName,
		"html_url": m.htmlURL,
	})
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(payload)),
	}, nil
}

func TestUpdateUpToDate(t *testing.T) {
	Version = "1.0.0"
	httpClient = &mockHTTPClient{tagName: "v1.0.0"}
	t.Cleanup(func() { Version = "dev"; httpClient = http.DefaultClient })

	var out bytes.Buffer
	if err := executeUpdate(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "up to date") {
		t.Errorf("expected up-to-date message, got: %q", out.String())
	}
}

func TestUpdateNewVersionAvailable(t *testing.T) {
	Version = "1.0.0"
	httpClient = &mockHTTPClient{tagName: "v1.1.0", htmlURL: "https://github.com/spw-m-riley/streep/releases/v1.1.0"}
	t.Cleanup(func() { Version = "dev"; httpClient = http.DefaultClient })

	var out bytes.Buffer
	if err := executeUpdate(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "v1.1.0") {
		t.Errorf("expected new version in output, got: %q", got)
	}
	if !strings.Contains(got, "brew upgrade") {
		t.Errorf("expected upgrade instructions, got: %q", got)
	}
}

func TestUpdateDevBuild(t *testing.T) {
	Version = "dev"
	var out bytes.Buffer
	if err := executeUpdate(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dev build") {
		t.Errorf("expected dev build message, got: %q", out.String())
	}
}
