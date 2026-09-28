package msgparity

import (
	"reflect"
	"sort"
	"testing"

	"golang.org/x/text/language"
)

// TestRequiredCategories checks requiredCategories against the CLDR
// cardinal-rule category sets documented at
// https://www.unicode.org/cldr/charts/45/supplemental/language_plural_rules.html
// for a language from each shape of rule: none (Japanese), two categories
// with a European cutoff (English), the Slavic one/few/many/other split
// (Polish, Russian), and the full six-category set (Arabic).
func TestRequiredCategories(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"ja", []string{"other"}},
		{"en", []string{"one", "other"}},
		{"pl", []string{"few", "many", "one", "other"}},
		{"ru", []string{"few", "many", "one", "other"}},
		{"ar", []string{"few", "many", "one", "other", "two", "zero"}},
	}
	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			tag := language.MustParse(tc.lang)
			got := requiredCategories(tag)
			var gotList []string
			for cat := range got {
				gotList = append(gotList, cat)
			}
			sort.Strings(gotList)
			if !reflect.DeepEqual(gotList, tc.want) {
				t.Errorf("requiredCategories(%s) = %v, want %v", tc.lang, gotList, tc.want)
			}
		})
	}
}

func TestRequiredCategories_AlwaysIncludesOther(t *testing.T) {
	for _, lang := range []string{"ja", "en", "pl", "ar", "lv", "cy", "fr"} {
		tag := language.MustParse(lang)
		if !requiredCategories(tag)["other"] {
			t.Errorf("requiredCategories(%s) does not include \"other\"", lang)
		}
	}
}
