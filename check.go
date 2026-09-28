package msgparity

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// localeSet is every message msgparity found for one locale, merged across
// however many files that locale was split across.
type localeSet struct {
	tag      language.Tag
	messages map[string]*i18n.Message // message ID -> first definition seen
	files    map[string][]string      // message ID -> every file that defines it, in load order
	paths    []string                 // every file loaded for this locale, sorted
}

// Check loads every message file input names (or, if input is a single
// directory, every message file directly inside it), and runs every
// msgparity rule against them relative to opts.Source (default "en").
func Check(input []string, opts Options) (Result, error) {
	source := opts.Source
	if source == "" {
		source = "en"
	}
	sourceTag, err := language.Parse(source)
	if err != nil {
		return Result{}, fmt.Errorf("invalid source locale %q: %w", source, err)
	}

	dir, names, err := resolveInputs(input)
	if err != nil {
		return Result{}, err
	}

	loaded, findings := loadFiles(dir, names)
	result := Result{Findings: findings}
	if len(loaded) == 0 {
		return result, nil
	}

	locales := map[string]*localeSet{}
	for _, f := range loaded {
		key := f.Locale.String()
		ls := locales[key]
		if ls == nil {
			ls = &localeSet{tag: f.Locale, messages: map[string]*i18n.Message{}, files: map[string][]string{}}
			locales[key] = ls
		}
		ls.paths = append(ls.paths, f.Path)
		for _, m := range f.Messages {
			ls.files[m.ID] = append(ls.files[m.ID], f.Path)
			if _, exists := ls.messages[m.ID]; !exists {
				ls.messages[m.ID] = m
			}
		}
	}

	for _, ls := range locales {
		for id, paths := range ls.files {
			if len(paths) > 1 {
				result.Findings = append(result.Findings, Finding{
					Rule:      RuleDuplicateID,
					Severity:  SeverityError,
					File:      strings.Join(paths, ", "),
					Locale:    ls.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("defined in %d files: %s", len(paths), strings.Join(paths, ", ")),
				})
			}
		}
	}

	srcLocale, ok := locales[sourceTag.String()]
	if !ok {
		return result, fmt.Errorf("no message file found for source locale %q (looked for tag %q among: %s)", source, sourceTag.String(), locKeys(locales))
	}

	// Fields the source message uses, and whether the source template
	// itself is well-formed, computed once and reused for every locale.
	type srcInfo struct {
		fields   map[string]bool
		forms    map[string]string
		isPlural bool
	}
	srcInfos := make(map[string]srcInfo, len(srcLocale.messages))
	for id, msg := range srcLocale.messages {
		forms := messageForms(msg)
		fields := map[string]bool{}
		srcPath := firstPath(srcLocale.files[id])
		for cat, text := range forms {
			f, err := templateFields(text, msg.LeftDelim, msg.RightDelim)
			if err != nil {
				result.Findings = append(result.Findings, Finding{
					Rule:      RuleTemplateSyntax,
					Severity:  SeverityError,
					File:      srcPath,
					Locale:    srcLocale.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("%s form: %s", cat, err),
				})
				continue
			}
			for k := range f {
				fields[k] = true
			}
		}
		srcInfos[id] = srcInfo{fields: fields, forms: forms, isPlural: len(forms) > 1}
	}

	for key, ls := range locales {
		if key == sourceTag.String() {
			continue
		}

		for id := range ls.messages {
			if _, inSource := srcLocale.messages[id]; !inSource {
				result.Findings = append(result.Findings, Finding{
					Rule:      RuleOrphanID,
					Severity:  SeverityWarning,
					File:      firstPath(ls.files[id]),
					Locale:    ls.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("message %q is not defined in the source locale %q", id, source),
				})
			}
		}

		for id, info := range srcInfos {
			transMsg, present := ls.messages[id]
			transPath := firstPath(ls.files[id])
			if !present {
				result.Findings = append(result.Findings, Finding{
					Rule:      RuleMissingID,
					Severity:  SeverityWarning,
					File:      strings.Join(ls.paths, ", "),
					Locale:    ls.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("message %q from the source locale has no translation for %q", id, ls.tag.String()),
				})
				continue
			}

			transForms := messageForms(transMsg)
			transFields := map[string]bool{}
			for cat, text := range transForms {
				f, err := templateFields(text, transMsg.LeftDelim, transMsg.RightDelim)
				if err != nil {
					result.Findings = append(result.Findings, Finding{
						Rule:      RuleTemplateSyntax,
						Severity:  SeverityError,
						File:      transPath,
						Locale:    ls.tag.String(),
						MessageID: id,
						Message:   fmt.Sprintf("%s form: %s", cat, err),
					})
					continue
				}
				for k := range f {
					transFields[k] = true
				}
			}

			var missing, extra []string
			for f := range info.fields {
				if !transFields[f] {
					missing = append(missing, f)
				}
			}
			for f := range transFields {
				if !info.fields[f] {
					extra = append(extra, f)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)
			if len(missing) > 0 {
				result.Findings = append(result.Findings, Finding{
					Rule:      RulePlaceholderMissing,
					Severity:  SeverityError,
					File:      transPath,
					Locale:    ls.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("translation drops %s used by the source message: %s", pluralize(len(missing), "placeholder"), joinFields(missing)),
				})
			}
			if len(extra) > 0 {
				result.Findings = append(result.Findings, Finding{
					Rule:      RulePlaceholderExtra,
					Severity:  SeverityError,
					File:      transPath,
					Locale:    ls.tag.String(),
					MessageID: id,
					Message:   fmt.Sprintf("translation references %s the source message does not use: %s", pluralize(len(extra), "placeholder"), joinFields(extra)),
				})
			}

			if info.isPlural {
				required := requiredCategories(ls.tag)
				present := map[string]bool{}
				for cat := range transForms {
					present[cat] = true
				}
				var missingCats, unusedCats []string
				for _, cat := range pluralCategories {
					if required[cat] && !present[cat] {
						missingCats = append(missingCats, cat)
					}
					if !required[cat] && present[cat] {
						unusedCats = append(unusedCats, cat)
					}
				}
				for _, cat := range missingCats {
					result.Findings = append(result.Findings, Finding{
						Rule:      RulePluralMissing,
						Severity:  SeverityError,
						File:      transPath,
						Locale:    ls.tag.String(),
						MessageID: id,
						Message:   fmt.Sprintf("missing plural category %q, which %s's CLDR cardinal rules require for this message", cat, ls.tag.String()),
					})
				}
				for _, cat := range unusedCats {
					result.Findings = append(result.Findings, Finding{
						Rule:      RulePluralUnused,
						Severity:  SeverityWarning,
						File:      transPath,
						Locale:    ls.tag.String(),
						MessageID: id,
						Message:   fmt.Sprintf("defines plural category %q, which %s's CLDR cardinal rules never select", cat, ls.tag.String()),
					})
				}
			}
		}
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Locale != b.Locale {
			return a.Locale < b.Locale
		}
		if a.MessageID != b.MessageID {
			return a.MessageID < b.MessageID
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Message < b.Message
	})

	return result, nil
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func joinFields(fields []string) string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = "{{." + f + "}}"
	}
	return strings.Join(out, ", ")
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func locKeys(locales map[string]*localeSet) string {
	keys := make([]string, 0, len(locales))
	for k := range locales {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
