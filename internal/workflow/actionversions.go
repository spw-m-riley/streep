package workflow

// DeprecatedActionVersions maps deprecated uses: references to recommended replacements.
var DeprecatedActionVersions = map[string]string{
	"actions/checkout@v1":          "actions/checkout@v4",
	"actions/checkout@v2":          "actions/checkout@v4",
	"actions/setup-node@v1":        "actions/setup-node@v4",
	"actions/setup-node@v2":        "actions/setup-node@v4",
	"actions/setup-go@v1":          "actions/setup-go@v5",
	"actions/setup-go@v2":          "actions/setup-go@v5",
	"actions/upload-artifact@v1":   "actions/upload-artifact@v4",
	"actions/upload-artifact@v2":   "actions/upload-artifact@v4",
	"actions/upload-artifact@v3":   "actions/upload-artifact@v4",
	"actions/download-artifact@v1": "actions/download-artifact@v4",
	"actions/download-artifact@v2": "actions/download-artifact@v4",
	"actions/download-artifact@v3": "actions/download-artifact@v4",
	"actions/cache@v1":             "actions/cache@v4",
	"actions/cache@v2":             "actions/cache@v4",
}
