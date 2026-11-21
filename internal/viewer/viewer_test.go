package viewer

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestReportDataStaysInsideJSONAndCSPHashesMatch(t *testing.T) {
	// A report is untrusted input, including its filenames and identity names.
	payload := `</script><script>window.untrusted=true</script>&{{JS}}` + "\u2028"
	text, _ := json.Marshal(payload)
	data := []byte(`{"schemaVersion":"1.0","authorizationModel":"kubernetes-rbac","scope":{"calculation":"incomplete"},"changes":[],"diagnostics":[],"beforeSource":` + string(text) + `,"comparedClasses":9007199254740993}`)
	page, err := Render(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(page, []byte(payload)) || bytes.Count(page, []byte("</script>")) != 3 {
		t.Fatal("report data escaped its JSON script")
	}
	contents := regexp.MustCompile(`(?s)<(?:script[^>]*|style)>(.*?)</(?:script|style)>`).FindAllSubmatch(page, -1)
	if len(contents) != 4 {
		t.Fatal("expected three inline scripts and one stylesheet")
	}
	for _, content := range contents {
		digest := sha256.Sum256(content[1])
		if !bytes.Contains(page, []byte("'sha256-"+base64.StdEncoding.EncodeToString(digest[:])+"'")) {
			t.Fatal("inline content hash does not match CSP")
		}
	}
	initial := regexp.MustCompile(`(?s)id="initial-report">(.*?)</script>`).FindSubmatch(page)[1]
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(initial, &decoded); err != nil {
		t.Fatal(err)
	}
	var source string
	if err := json.Unmarshal(decoded["beforeSource"], &source); err != nil || source != payload {
		t.Fatal("report text was lost while escaping HTML")
	}
	if string(decoded["comparedClasses"]) != "9007199254740993" {
		t.Fatal("report encoding rounded a number before browser validation")
	}
	if bytes.Contains(page, []byte("\r")) || !bytes.Contains(page, []byte("connect-src 'none'")) {
		t.Fatal("offline content policy or normalized newlines missing")
	}
}

func TestLoadHandlesBOMEmptyAndOversizeReports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	for _, content := range [][]byte{nil, []byte(" \r\n\t"), {0xef, 0xbb, 0xbf}} {
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("empty report accepted")
		}
	}
	if err := os.WriteFile(path, []byte("\ufeff{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := Load(path); err != nil || string(data) != "{}" {
		t.Fatalf("BOM was not normalized: %s %v", data, err)
	}
	if err := os.Truncate(path, MaxReportBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("oversize report accepted")
	}
}

func TestRenderRejectsWrongReportEnvelope(t *testing.T) {
	for _, data := range []string{
		`not JSON`, `null`, `{}`, `[]`,
		`{"schemaVersion":"1.0","authorizationModel":"kubernetes-rbac","scope":{"calculation":"complete"},"changes":null,"diagnostics":[]}`,
		`{"schemaVersion":"2.0","authorizationModel":"kubernetes-rbac","scope":{"calculation":"complete"},"changes":[],"diagnostics":[]}`,
		`{"schemaVersion":"1.0","authorizationModel":"kubernetes-rbac","scope":{"calculation":"guess"},"changes":[],"diagnostics":[]}`,
	} {
		if _, err := Render([]byte(data)); err == nil {
			t.Fatalf("invalid report accepted: %s", data)
		}
	}
}
