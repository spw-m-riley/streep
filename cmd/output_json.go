package cmd

import (
	"encoding/json"
	"io"
	"strings"
)

type commandJSONResult struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Output string `json:"output,omitempty"`
}

func splitJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonMode
}

func splitPassthroughArgs(args []string) (primary []string, passthrough []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeWrappedJSON(out io.Writer, text string, runErr error) error {
	payload := commandJSONResult{
		OK:     runErr == nil,
		Output: strings.TrimRight(text, "\n"),
	}
	if runErr != nil {
		payload.Error = runErr.Error()
	}
	return writeJSON(out, payload)
}
