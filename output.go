package msgparity

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format selects a Result output encoding.
type Format string

// Supported formats.
const (
	FormatText   Format = "text"
	FormatJSON   Format = "json"
	FormatGitHub Format = "github"
)

// WriteResult writes a Result in the given format to w.
func WriteResult(w io.Writer, result Result, format Format) error {
	switch format {
	case "", FormatText:
		return writeText(w, result)
	case FormatJSON:
		return writeJSON(w, result)
	case FormatGitHub:
		return writeGitHub(w, result)
	default:
		return fmt.Errorf("unknown format %q (want text, json or github)", format)
	}
}

func writeText(w io.Writer, result Result) error {
	if len(result.Findings) == 0 {
		_, err := fmt.Fprintln(w, "msgparity: no findings")
		return err
	}
	for _, f := range result.Findings {
		if _, err := fmt.Fprintln(w, f.String()); err != nil {
			return err
		}
	}
	errs, warns := 0, 0
	for _, f := range result.Findings {
		if f.Severity == SeverityError {
			errs++
		} else {
			warns++
		}
	}
	_, err := fmt.Fprintf(w, "msgparity: %d error(s), %d warning(s)\n", errs, warns)
	return err
}

// jsonFinding is the --format json shape for one Finding: stable field
// names, independent of the in-memory Finding layout.
type jsonFinding struct {
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	File      string `json:"file,omitempty"`
	Locale    string `json:"locale,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Message   string `json:"message"`
}

func writeJSON(w io.Writer, result Result) error {
	out := make([]jsonFinding, 0, len(result.Findings))
	for _, f := range result.Findings {
		out = append(out, jsonFinding{
			Rule:      f.Rule,
			Severity:  string(f.Severity),
			File:      f.File,
			Locale:    f.Locale,
			MessageID: f.MessageID,
			Message:   f.Message,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// writeGitHub writes GitHub Actions workflow command annotations:
// https://docs.github.com/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message
func writeGitHub(w io.Writer, result Result) error {
	for _, f := range result.Findings {
		level := "warning"
		if f.Severity == SeverityError {
			level = "error"
		}
		title := "msgparity " + f.Rule
		if f.Locale != "" {
			title += " (" + f.Locale + ")"
		}
		if _, err := fmt.Fprintf(w, "::%s file=%s,title=%s::%s\n", level, f.File, title, f.Message); err != nil {
			return err
		}
	}
	return nil
}
