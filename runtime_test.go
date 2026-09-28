package msgparity

// This file cross-checks msgparity's verdicts against go-i18n's own runtime
// behavior: every fixture msgparity passes must actually localize cleanly
// through go-i18n for every locale, message, and plural count exercised;
// every fixture msgparity fails on a placeholder or plural rule must be
// shown, by actually calling i18n.Localizer.Localize, to either error or
// silently render the wrong thing.

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// sampleCounts are the plural counts specified for the runtime cross-check:
// 0 and 1 hit "one"/"zero" boundaries, 2 hits "two"/"few", 5/25/100 hit
// "many"/"other" depending on language, 21/22 hit the Slavic "n%10" rule's
// interaction with the "n%100 in 11..14" exception.
var sampleCounts = []int{0, 1, 2, 5, 21, 22, 25, 100}

func newRuntimeBundle(t *testing.T, dir string, locales ...string) *i18n.Bundle {
	t.Helper()
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	for _, loc := range locales {
		found := false
		for _, ext := range []string{"toml", "json", "yaml", "yml"} {
			path := dir + "/" + loc + "." + ext
			if _, err := bundle.LoadMessageFile(path); err == nil {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no message file for locale %q in %s", loc, dir)
		}
	}
	return bundle
}

// TestRuntimeCrossCheck_CleanFixturesLocalizeWithoutError runs every message
// in testdata/formats through the real go-i18n localizer, in all three file
// formats, for every locale and every sample count, and requires no error.
// msgparity passes this fixture set (proven by TestCheck_FormatsFixtureIsClean
// in formats_test.go); this test is the other half of the claim: go-i18n
// agrees there is nothing wrong with it.
func TestRuntimeCrossCheck_CleanFixturesLocalizeWithoutError(t *testing.T) {
	for _, format := range []string{"toml", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			dir := "testdata/formats/" + format
			bundle := newRuntimeBundle(t, dir, "en", "es", "pl", "ar", "ja")
			for _, locale := range []string{"es", "pl", "ar", "ja"} {
				loc := i18n.NewLocalizer(bundle, locale)
				for _, count := range sampleCounts {
					// Greeting is not plural in the source (only "other"
					// is defined): a realistic caller would not attach a
					// PluralCount to it, and go-i18n only runs CLDR
					// plural-category resolution when PluralCount is
					// non-nil.
					msg, err := loc.Localize(&i18n.LocalizeConfig{
						MessageID:    "Greeting",
						TemplateData: map[string]interface{}{"Name": "Ana", "Count": count},
					})
					if err != nil {
						t.Errorf("%s/%s Greeting count=%d: unexpected error: %v", format, locale, count, err)
					}
					if msg == "" {
						t.Errorf("%s/%s Greeting count=%d: empty message", format, locale, count)
					}

					// PersonCats is plural in the source (one + other),
					// which is exactly what requiredCategories requires
					// the clean fixture to supply per language.
					msg, err = loc.Localize(&i18n.LocalizeConfig{
						MessageID:    "PersonCats",
						PluralCount:  count,
						TemplateData: map[string]interface{}{"Name": "Ana", "Count": count},
					})
					if err != nil {
						t.Errorf("%s/%s PersonCats count=%d: unexpected error: %v", format, locale, count, err)
					}
					if msg == "" {
						t.Errorf("%s/%s PersonCats count=%d: empty message", format, locale, count)
					}
				}
			}
		})
	}
}

// TestRuntimeCrossCheck_PlaceholderDropRendersSilentlyWrong reproduces the
// build brief's premise repro: a Spanish translation that drops both source
// placeholders. msgparity fails this fixture (TestCheck_PlaceholderMissing).
// go-i18n does not error — it renders the translation text verbatim, minus
// the data the caller passed, which is the silent-wrongness failure mode
// `goi18n merge` cannot catch because there is no error for it to catch.
func TestRuntimeCrossCheck_PlaceholderDropRendersSilentlyWrong(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[Greeting]
other = "Hello, {{.Name}}! You have {{.Count}} messages."
`,
		"es.toml": `[Greeting]
other = "Hola!"
`,
	})
	bundle := newRuntimeBundle(t, dir, "en", "es")
	loc := i18n.NewLocalizer(bundle, "es")
	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    "Greeting",
		PluralCount:  5, // "other" in both en and es, isolating the placeholder drop
		TemplateData: map[string]interface{}{"Name": "Ana", "Count": 5},
	})
	if err != nil {
		t.Fatalf("go-i18n unexpectedly errored (expected a silent, wrong render): %v", err)
	}
	if strings.Contains(msg, "Ana") || strings.Contains(msg, "5") {
		t.Fatalf("expected the dropped placeholders to be silently missing from the render, got %q", msg)
	}
}

// TestRuntimeCrossCheck_PluralIncompleteErrorsAtEveryIntegerCount
// reproduces the Polish repro: a translation supplying only "other" for a
// plural message. msgparity fails this fixture
// (TestCheck_PluralMissing_Polish). go-i18n errors for every one of the
// eight specified sample counts, because none of them fall into Polish's
// "other" category for integers — proving the premise that this reaches
// runtime uncaught, not merely in a rare edge case.
func TestRuntimeCrossCheck_PluralIncompleteErrorsAtEveryIntegerCount(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[PersonCats]
one = "{{.Name}} has {{.Count}} cat."
other = "{{.Name}} has {{.Count}} cats."
`,
		"pl.toml": `[PersonCats]
other = "{{.Name}} ma {{.Count}} kotów."
`,
	})
	bundle := newRuntimeBundle(t, dir, "en", "pl")
	loc := i18n.NewLocalizer(bundle, "pl")
	for _, count := range sampleCounts {
		_, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID:    "PersonCats",
			PluralCount:  count,
			TemplateData: map[string]interface{}{"Name": "Ana", "Count": count},
		})
		if err == nil {
			t.Errorf("count=%d: want a \"has no plural form\" error from go-i18n, got none", count)
		} else if !strings.Contains(err.Error(), "has no plural form") {
			t.Errorf("count=%d: got error %v, want a \"has no plural form\" error", count, err)
		}
	}
}

// TestRuntimeCrossCheck_ExtraPlaceholderFailsOrRendersWrong verifies the
// build brief's claim about typo'd placeholder names, which turns out to be
// TemplateData-shape dependent: go-i18n hard-fails when TemplateData is a
// struct (text/template's field lookup on a struct returns an error for an
// unknown field) but silently renders "<no value>" when TemplateData is a
// map (text/template's map lookup for a missing key is not an error). Both
// are real bugs; msgparity flags both as RulePlaceholderExtra
// (TestCheck_PlaceholderExtra) without needing to know which TemplateData
// shape the caller will use.
func TestRuntimeCrossCheck_ExtraPlaceholderFailsOrRendersWrong(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"en.toml": `[PersonCats]
one = "{{.Name}} has {{.Count}} cat."
other = "{{.Name}} has {{.Count}} cats."
`,
		"es.toml": `[PersonCats]
one = "{{.Nmae}} tiene {{.Count}} gato."
other = "{{.Nmae}} tiene {{.Count}} gatos."
`,
	})
	bundle := newRuntimeBundle(t, dir, "en", "es")
	loc := i18n.NewLocalizer(bundle, "es")

	t.Run("map TemplateData renders silently wrong", func(t *testing.T) {
		msg, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID:    "PersonCats",
			PluralCount:  1,
			TemplateData: map[string]interface{}{"Name": "Ana", "Count": 1},
		})
		if err != nil {
			t.Fatalf("unexpected error with map TemplateData: %v", err)
		}
		if !strings.Contains(msg, "<no value>") {
			t.Fatalf("want \"<no value>\" for the typo'd field with map TemplateData, got %q", msg)
		}
	})

	t.Run("struct TemplateData errors", func(t *testing.T) {
		type Data struct {
			Name  string
			Count int
		}
		_, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID:    "PersonCats",
			PluralCount:  1,
			TemplateData: Data{Name: "Ana", Count: 1},
		})
		if err == nil {
			t.Fatal("want an error for the typo'd field with struct TemplateData")
		}
		if !strings.Contains(err.Error(), "can't evaluate field") {
			t.Errorf("got error %v, want a \"can't evaluate field\" error", err)
		}
	})
}
