package msgparity

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFiles writes name -> content pairs into a fresh temp directory and
// returns the directory.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func findingsWithRule(findings []Finding, rule string) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func TestCheck_PlaceholderMissing(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Greeting]
other = "Hello, {{.Name}}! You have {{.Count}} messages."
`,
		"es.toml": `[Greeting]
other = "Hola!"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RulePlaceholderMissing)
	if len(found) != 1 {
		t.Fatalf("want 1 placeholder-missing finding, got %d: %+v", len(found), result.Findings)
	}
	if found[0].Severity != SeverityError {
		t.Errorf("severity = %s, want error", found[0].Severity)
	}
}

func TestCheck_PlaceholderMissing_Clean(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Greeting]
other = "Hello, {{.Name}}! You have {{.Count}} messages."
`,
		"es.toml": `[Greeting]
other = "Hola, {{.Name}}! Tienes {{.Count}} mensajes."
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("want no findings on a clean translation, got %+v", result.Findings)
	}
}

func TestCheck_PlaceholderExtra(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Greeting]
other = "Hello, {{.Name}}!"
`,
		"es.toml": `[Greeting]
other = "Hola, {{.Nmae}}!"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RulePlaceholderExtra)
	if len(found) != 1 {
		t.Fatalf("want 1 placeholder-extra finding, got %d: %+v", len(found), result.Findings)
	}
}

func TestCheck_PluralMissing_Polish(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[PersonCats]
one = "{{.Name}} has {{.Count}} cat."
other = "{{.Name}} has {{.Count}} cats."
`,
		"pl.toml": `[PersonCats]
other = "{{.Name}} ma {{.Count}} kotów."
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RulePluralMissing)
	want := map[string]bool{"one": true, "few": true, "many": true}
	if len(found) != len(want) {
		t.Fatalf("want %d plural-missing findings, got %d: %+v", len(want), len(found), result.Findings)
	}
	for _, f := range found {
		if f.Severity != SeverityError {
			t.Errorf("severity = %s, want error", f.Severity)
		}
	}
}

func TestCheck_PluralMissing_Clean(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[PersonCats]
one = "{{.Name}} has {{.Count}} cat."
other = "{{.Name}} has {{.Count}} cats."
`,
		"pl.toml": `[PersonCats]
one = "{{.Name}} ma {{.Count}} kota."
few = "{{.Name}} ma {{.Count}} koty."
many = "{{.Name}} ma {{.Count}} kotów."
other = "{{.Name}} ma {{.Count}} kota."
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("want no findings for a plural-complete Polish translation, got %+v", result.Findings)
	}
}

func TestCheck_PluralMissing_OnlyForPluralSourceMessages(t *testing.T) {
	// A message the source only ever defines as "other" is not plural in
	// intent; a translation with only "other" must not be flagged even
	// though Polish's CLDR rules include one/few/many.
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Static]
other = "Hello there."
`,
		"pl.toml": `[Static]
other = "Witaj."
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("want no findings for a non-plural message, got %+v", result.Findings)
	}
}

func TestCheck_PluralUnused_Warning(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[PersonCats]
one = "{{.Name}} has {{.Count}} cat."
other = "{{.Name}} has {{.Count}} cats."
`,
		"pl.toml": `[PersonCats]
one = "{{.Name}} ma {{.Count}} kota."
two = "{{.Name}} ma dwa koty."
few = "{{.Name}} ma {{.Count}} koty."
many = "{{.Name}} ma {{.Count}} kotów."
other = "{{.Name}} ma {{.Count}} kota."
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RulePluralUnused)
	if len(found) != 1 {
		t.Fatalf("want 1 plural-unused finding for pl's unused 'two' category, got %d: %+v", len(found), result.Findings)
	}
	if found[0].Severity != SeverityWarning {
		t.Errorf("severity = %s, want warning", found[0].Severity)
	}
}

func TestCheck_MissingID(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Hello]
other = "Hello"
[Goodbye]
other = "Goodbye"
`,
		"es.toml": `[Hello]
other = "Hola"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RuleMissingID)
	if len(found) != 1 || found[0].Severity != SeverityWarning {
		t.Fatalf("want 1 missing-id warning, got %+v", result.Findings)
	}
}

func TestCheck_OrphanID(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Hello]
other = "Hello"
`,
		"es.toml": `[Hello]
other = "Hola"
[Leftover]
other = "Sobrante"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RuleOrphanID)
	if len(found) != 1 || found[0].Severity != SeverityWarning {
		t.Fatalf("want 1 orphan-id warning, got %+v", result.Findings)
	}
}

func TestCheck_DuplicateID(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.en.toml": `[Hello]
other = "Hello"
`,
		"b.en.toml": `[Hello]
other = "Hello again"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RuleDuplicateID)
	if len(found) != 1 || found[0].Severity != SeverityError {
		t.Fatalf("want 1 duplicate-id error, got %+v", result.Findings)
	}
}

func TestCheck_TemplateSyntax(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Broken]
other = "Hello {{.Name"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	found := findingsWithRule(result.Findings, RuleTemplateSyntax)
	if len(found) != 1 || found[0].Severity != SeverityError {
		t.Fatalf("want 1 template-syntax error, got %+v", result.Findings)
	}
}

func TestCheck_JSONAndYAML(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
	}{
		{
			name: "json",
			files: map[string]string{
				"en.json": `{"Greeting": {"other": "Hello, {{.Name}}!"}}`,
				"es.json": `{"Greeting": {"other": "Hola!"}}`,
			},
		},
		{
			name: "yaml",
			files: map[string]string{
				"en.yaml": "Greeting:\n  other: \"Hello, {{.Name}}!\"\n",
				"es.yaml": "Greeting:\n  other: \"Hola!\"\n",
			},
		},
		{
			name: "yml",
			files: map[string]string{
				"en.yml": "Greeting:\n  other: \"Hello, {{.Name}}!\"\n",
				"es.yml": "Greeting:\n  other: \"Hola!\"\n",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, tc.files)
			result, err := Check([]string{dir}, Options{})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			found := findingsWithRule(result.Findings, RulePlaceholderMissing)
			if len(found) != 1 {
				t.Fatalf("want 1 placeholder-missing finding, got %d: %+v", len(found), result.Findings)
			}
		})
	}
}

func TestCheck_ExplicitFileList(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Greeting]
other = "Hello, {{.Name}}!"
`,
		"es.toml": `[Greeting]
other = "Hola, {{.Name}}!"
`,
		"ignored.toml": `not even valid, and should never be read`,
	})
	result, err := Check([]string{
		filepath.Join(dir, "en.toml"),
		filepath.Join(dir, "es.toml"),
	}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("want no findings, got %+v", result.Findings)
	}
}

func TestCheck_NoSourceLocale(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"es.toml": `[Hello]
other = "Hola"
`,
	})
	_, err := Check([]string{dir}, Options{Source: "en"})
	if err == nil {
		t.Fatal("want an error when the source locale has no file")
	}
}

func TestCheck_LoadError(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.json": `{not valid json`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Rule != "load-error" {
		t.Fatalf("want 1 load-error finding, got %+v", result.Findings)
	}
}

func TestCheck_UnresolvableLocale(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"messages.toml": `[Hello]
other = "Hello"
`,
	})
	result, err := Check([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Rule != "load-error" {
		t.Fatalf("want 1 load-error finding for a filename with no locale segment, got %+v", result.Findings)
	}
}
