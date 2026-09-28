package msgparity

import (
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

// pluralSampleInts is sampled through golang.org/x/text/feature/plural's
// CLDR cardinal rules to discover which plural categories a language can
// select for an integer count. CLDR plural rules are defined in terms of a
// number's last one or two decimal digits (n % 10, n % 100), so 0..199
// covers every modulus pattern in the specification actually in use, plus a
// handful of digit-repetition edge cases (11-14, 111-114) some Slavic and
// Baltic languages special-case.
var pluralSampleInts = func() []int {
	ints := make([]int, 0, 210)
	for i := 0; i <= 199; i++ {
		ints = append(ints, i)
	}
	ints = append(ints, 1000, 10000, 100000)
	return ints
}()

// formName maps a plural.Form to the lowercase CLDR category name go-i18n
// uses for the corresponding Message field.
func formName(f plural.Form) string {
	switch f {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	case plural.Other:
		return "other"
	default:
		return "other"
	}
}

// requiredCategories returns the CLDR plural categories tag's cardinal
// rules can select for an integer count, derived by sampling
// plural.Cardinal.MatchPlural (golang.org/x/text's CLDR-generated cardinal
// plural rules — see the package doc comment on why this, not go-i18n's own
// rules, is what msgparity checks against). "other" is always included: it
// is CLDR's mandatory fallback category, and go-i18n errors if even it is
// missing from a message that has any other plural form.
func requiredCategories(tag language.Tag) map[string]bool {
	forms := map[string]bool{"other": true}
	for _, n := range pluralSampleInts {
		f := plural.Cardinal.MatchPlural(tag, n, 0, 0, 0, 0)
		forms[formName(f)] = true
	}
	return forms
}
