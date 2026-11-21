// Package viewer produces a self-contained, offline report viewer.
package viewer

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jintilm/GrantDiff/docs"
)

const MaxReportBytes = 128 << 20

//go:embed index.html style.css app.js
var assets embed.FS

func Load(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxReportBytes {
		return nil, fmt.Errorf("viewer report exceeds %d bytes", MaxReportBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxReportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxReportBytes {
		return nil, fmt.Errorf("viewer report exceeds %d bytes", MaxReportBytes)
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("viewer input report is empty")
	}
	return data, nil
}

func Render(data []byte) ([]byte, error) {
	encoded := []byte("null")
	if len(data) > 0 {
		// Check the envelope here. The browser validates the full schema for
		// embedded reports and files selected through the same interface.
		var report struct {
			SchemaVersion      string
			AuthorizationModel string
			Changes            json.RawMessage
			Diagnostics        json.RawMessage
			Scope              struct{ Calculation string }
		}
		if err := json.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("invalid diff report: %w", err)
		}
		if report.SchemaVersion != "1.0" || report.AuthorizationModel != "kubernetes-rbac" || !bytes.HasPrefix(report.Changes, []byte("[")) || !bytes.HasPrefix(report.Diagnostics, []byte("[")) || report.Scope.Calculation != "complete" && report.Scope.Calculation != "incomplete" {
			return nil, fmt.Errorf("viewer requires a GrantDiff schema 1.0 diff report")
		}
		// RawMessage preserves number precision. Marshal escapes HTML-sensitive
		// characters before the report enters a script element.
		var err error
		encoded, err = json.Marshal(json.RawMessage(data))
		if err != nil {
			return nil, err
		}
	}
	page, _ := assets.ReadFile("index.html")
	css, _ := assets.ReadFile("style.css")
	js, _ := assets.ReadFile("app.js")
	// HTML parsers normalize CRLF before checking inline content hashes.
	css = bytes.ReplaceAll(css, []byte("\r\n"), []byte("\n"))
	js = bytes.ReplaceAll(js, []byte("\r\n"), []byte("\n"))
	schema := bytes.ReplaceAll(docs.ReportSchema, []byte("\r\n"), []byte("\n"))
	page = bytes.ReplaceAll(page, []byte("\r\n"), []byte("\n"))
	hash := func(value []byte) string {
		digest := sha256.Sum256(value)
		return "'sha256-" + base64.StdEncoding.EncodeToString(digest[:]) + "'"
	}
	csp := "default-src 'none'; script-src " + hash(js) + " " + hash(encoded) + " " + hash(schema) + "; style-src " + hash(css) + "; connect-src 'none'; base-uri 'none'; form-action 'none'"
	result := strings.NewReplacer("{{CSP}}", csp, "{{CSS}}", string(css), "{{REPORT}}", string(encoded), "{{SCHEMA}}", string(schema), "{{JS}}", string(js)).Replace(string(page))
	return []byte(result), nil
}
