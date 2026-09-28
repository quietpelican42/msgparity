// Package msgparity checks go-i18n (github.com/nicksnyder/go-i18n/v2) message
// files for two classes of bug that reach production without a build
// failure: a translation that drops or mistypes a template placeholder the
// source message uses, and a translation that is missing a CLDR plural
// category its own language requires.
//
// Neither is caught by `goi18n merge`, which only tracks which message IDs
// have been translated, not what a translation's template actions or plural
// forms are. Both are caught by go-i18n itself, but only at the moment a
// specific plural count or template field is actually requested at runtime —
// which for a rarely-hit language or count can be long after a translation
// ships.
package msgparity

import "fmt"

// Severity classifies how serious a Finding is.
type Severity string

// Severities, in increasing order of importance.
const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Rule names. Each corresponds to one check msgparity runs.
const (
	// RulePlaceholderMissing fires when a translation's plural forms, taken
	// together, reference fewer template fields than the source message.
	RulePlaceholderMissing = "placeholder-missing"
	// RulePlaceholderExtra fires when a translation references a template
	// field the source message does not use — usually a typo of a real
	// field name, which go-i18n either renders as "<no value>" (map
	// TemplateData) or fails to render at all (struct TemplateData).
	RulePlaceholderExtra = "placeholder-extra"
	// RulePlaceholderMissingForm fires when one specific plural form of a
	// translation drops a field every one of the source message's own
	// forms uses (excluding Options.CountField), even though the
	// translation's other forms keep it — a bug RulePlaceholderMissing
	// cannot see because it only compares the union of fields across all
	// of a translation's forms.
	RulePlaceholderMissingForm = "placeholder-missing-form"
	// RuleTemplateSyntax fires when a plural form fails to parse as a Go
	// template.
	RuleTemplateSyntax = "template-syntax"
	// RulePluralMissing fires when a translation is missing a CLDR plural
	// category its language requires, for a message the source marks as
	// plural (more than one non-empty plural form).
	RulePluralMissing = "plural-missing"
	// RulePluralUnused fires when a translation defines a plural category
	// its language's CLDR rules never select. go-i18n never renders it; it
	// is dead text, not a bug, hence a warning rather than an error.
	RulePluralUnused = "plural-unused"
	// RuleMissingID fires when a message defined in the source locale has
	// no entry at all in another locale.
	RuleMissingID = "missing-id"
	// RuleOrphanID fires when a locale defines a message ID the source
	// locale does not have.
	RuleOrphanID = "orphan-id"
	// RuleDuplicateID fires when the same message ID is defined in more
	// than one file for the same locale.
	RuleDuplicateID = "duplicate-id"
)

// Finding is one problem msgparity detected in a set of message files.
type Finding struct {
	// Rule is the check that produced this finding (one of the Rule*
	// constants).
	Rule string
	// Severity is how serious the finding is.
	Severity Severity
	// File is the message file the finding is about, relative to the
	// checked directory when a directory was given.
	File string
	// Locale is the BCP 47 language tag of the file the finding is about,
	// as go-i18n itself derives it from the filename.
	Locale string
	// MessageID is the message the finding is about. Empty for file-wide
	// findings such as a load error.
	MessageID string
	// Message is a human-readable description of the problem.
	Message string
}

// String formats a Finding the way the text formatter prints it.
func (f Finding) String() string {
	loc := f.Locale
	if loc == "" {
		loc = "-"
	}
	id := f.MessageID
	if id == "" {
		id = "-"
	}
	return fmt.Sprintf("%s: [%s] %s: %s: %s: %s", f.File, loc, id, f.Rule, f.Severity, f.Message)
}

// Options configures a Check run.
type Options struct {
	// Source is the BCP 47 tag (as go-i18n parses it from a filename, e.g.
	// "en") of the locale every other locale is checked against. Defaults
	// to "en" when empty.
	Source string
	// CountField is the TemplateData field name RulePlaceholderMissingForm
	// treats as the plural count, and so does not require in every plural
	// form. Defaults to "Count" when empty — the field name go-i18n's own
	// README example uses alongside PluralCount, and the one every
	// msgparity fixture and the README's own examples use. It is
	// configurable because this is an application-level naming
	// convention, not something go-i18n enforces: go-i18n itself only
	// auto-populates TemplateData when the caller leaves it nil, and does
	// so under the key "PluralCount", not "Count" (see localizer.go in
	// nicksnyder/go-i18n/v2/i18n). A project that follows that fallback,
	// or names its own count field something else, needs this flag.
	CountField string
}

// Result is the outcome of checking one set of message files.
type Result struct {
	// Findings holds every problem msgparity found, in a stable order:
	// grouped by file, then by message ID.
	Findings []Finding
}

// HasErrors reports whether any finding in the result is an error (as
// opposed to a warning).
func (r Result) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}
