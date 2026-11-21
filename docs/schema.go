// Package docs embeds the report contract for the offline viewer.
package docs

import _ "embed"

//go:embed report.schema.json
var ReportSchema []byte
