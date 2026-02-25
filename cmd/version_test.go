package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	Version = "1.2.3"
	Commit = "abc1234"
	Date = "2026-01-01"
	t.Cleanup(func() { Version = "dev"; Commit = "none"; Date = "unknown" })

	var out bytes.Buffer
	if err := Execute([]string{"version"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"1.2.3", "abc1234", "2026-01-01"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in version output, got: %q", want, got)
		}
	}
}
