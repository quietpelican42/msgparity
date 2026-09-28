package msgparity

import "testing"

// TestCheck_FormatsFixtureIsClean runs the same message set, written in
// TOML, JSON and YAML, through Check and requires zero findings from all
// three. It is the "clean negative" for every rule at once, and it proves
// msgparity's rule logic does not depend on which format the messages
// arrived in — only load.go's per-format parsing differs.
func TestCheck_FormatsFixtureIsClean(t *testing.T) {
	for _, format := range []string{"toml", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			result, err := Check([]string{"testdata/formats/" + format}, Options{})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Errorf("want no findings, got %+v", result.Findings)
			}
		})
	}
}
