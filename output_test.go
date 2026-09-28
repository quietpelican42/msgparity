package msgparity

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleResult() Result {
	return Result{Findings: []Finding{
		{Rule: RulePlaceholderMissing, Severity: SeverityError, File: "es.toml", Locale: "es", MessageID: "Greeting", Message: "translation drops 1 placeholder used by the source message: {{.Name}}"},
		{Rule: RuleMissingID, Severity: SeverityWarning, File: "es.toml", Locale: "es", MessageID: "Goodbye", Message: "message \"Goodbye\" from the source locale has no translation for \"es\""},
	}}
}

func TestWriteResult_Text(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatText); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "placeholder-missing") || !strings.Contains(out, "missing-id") {
		t.Errorf("text output missing expected rules: %s", out)
	}
	if !strings.Contains(out, "1 error(s), 1 warning(s)") {
		t.Errorf("text output missing summary line: %s", out)
	}
}

func TestWriteResult_TextClean(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, Result{}, FormatText); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	if got := buf.String(); got != "msgparity: no findings\n" {
		t.Errorf("got %q", got)
	}
}

func TestWriteResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatJSON); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	var findings []jsonFinding
	if err := json.Unmarshal(buf.Bytes(), &findings); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(findings) != 2 {
		t.Fatalf("want 2 findings, got %d", len(findings))
	}
	if findings[0].Rule != RulePlaceholderMissing || findings[0].Severity != "error" {
		t.Errorf("unexpected finding: %+v", findings[0])
	}
}

func TestWriteResult_GitHub(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatGitHub); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "::error file=es.toml,title=msgparity placeholder-missing (es)::") {
		t.Errorf("missing error annotation: %s", out)
	}
	if !strings.Contains(out, "::warning file=es.toml,title=msgparity missing-id (es)::") {
		t.Errorf("missing warning annotation: %s", out)
	}
}

func TestWriteResult_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), Format("bogus")); err == nil {
		t.Fatal("want an error for an unknown format")
	}
}

func TestResult_HasErrors(t *testing.T) {
	if !sampleResult().HasErrors() {
		t.Error("want HasErrors true when a Finding has SeverityError")
	}
	warnOnly := Result{Findings: []Finding{{Severity: SeverityWarning}}}
	if warnOnly.HasErrors() {
		t.Error("want HasErrors false when every Finding is a warning")
	}
}
