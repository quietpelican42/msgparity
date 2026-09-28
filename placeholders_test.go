package msgparity

import (
	"reflect"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func TestTemplateFields(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"no fields", "Hello there.", nil},
		{"one field", "Hello, {{.Name}}!", []string{"Name"}},
		{"two fields", "{{.Name}} has {{.Count}} cats.", []string{"Count", "Name"}},
		{"repeated field counts once", "{{.Name}}, is that you, {{.Name}}?", []string{"Name"}},
		{"nested field takes root ident", "{{.User.Name}}", []string{"User"}},
		{"field inside if", "{{if .Admin}}Hello, {{.Name}}{{end}}", []string{"Admin", "Name"}},
		{"field inside range", "{{range .Items}}{{.}}{{end}} for {{.Owner}}", []string{"Items", "Owner"}},
		{"field inside with rebinds dot", "{{with .Profile}}{{.Bio}}{{end}}", []string{"Profile"}},
		{"field as function arg", "{{printf \"%s\" .Name}}", []string{"Name"}},
		{"local variable is not a field", "{{$x := .Name}}{{$x}}", []string{"Name"}},
		{"pipeline", "{{.Name | printf \"%s\"}}", []string{"Name"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TemplateFields(tc.text, "", "")
			if err != nil {
				t.Fatalf("TemplateFields(%q): %v", tc.text, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("TemplateFields(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestTemplateFields_SyntaxError(t *testing.T) {
	_, err := TemplateFields("Hello {{.Name", "", "")
	if err == nil {
		t.Fatal("want an error for an unclosed action")
	}
}

func TestTemplateFields_CustomDelims(t *testing.T) {
	got, err := TemplateFields("Hello, [[.Name]]!", "[[", "]]")
	if err != nil {
		t.Fatalf("TemplateFields: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"Name"}) {
		t.Errorf("got %v, want [Name]", got)
	}
}

func TestMessageForms(t *testing.T) {
	m := &i18n.Message{One: "one text", Other: "other text"}
	forms := messageForms(m)
	if len(forms) != 2 {
		t.Fatalf("want 2 forms, got %d: %v", len(forms), forms)
	}
	if forms["one"] != "one text" || forms["other"] != "other text" {
		t.Errorf("unexpected forms: %v", forms)
	}
}
