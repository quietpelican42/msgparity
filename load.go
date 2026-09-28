package msgparity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// unmarshalFuncs registers every non-JSON format go-i18n's own
// ParseMessageFileBytes supports out of the box (JSON is built in). YAML is
// registered under both extensions go-i18n's filename convention can
// produce.
var unmarshalFuncs = map[string]i18n.UnmarshalFunc{
	"toml": toml.Unmarshal,
	"yaml": yaml.Unmarshal,
	"yml":  yaml.Unmarshal,
}

// loadedFile is one parsed message file.
type loadedFile struct {
	// Path is the path as given on the command line (or, for a directory
	// argument, relative to it).
	Path string
	// Locale is the BCP 47 tag go-i18n's own filename parser derived for
	// this file. Never the zero language.Tag: an unparsable locale segment
	// makes the file unusable to go-i18n itself, so it is reported as a
	// load error instead of silently grouped under "und".
	Locale language.Tag
	// Messages are the messages go-i18n's own decoder parsed from this
	// file, in file order.
	Messages []*i18n.Message
}

var messageFileExts = map[string]bool{
	"json": true,
	"toml": true,
	"yaml": true,
	"yml":  true,
}

// resolveInputs expands a single directory argument into the message files
// directly inside it (go-i18n message files are not nested), or, given one
// or more file arguments, uses them as-is. It returns paths as they should
// be reported: relative to dir for a directory argument, exactly as given
// otherwise.
func resolveInputs(inputs []string) (dir string, files []string, err error) {
	if len(inputs) == 0 {
		return "", nil, fmt.Errorf("no input given")
	}
	if len(inputs) == 1 {
		info, statErr := os.Stat(inputs[0])
		if statErr != nil {
			return "", nil, fmt.Errorf("%s: %w", inputs[0], statErr)
		}
		if info.IsDir() {
			entries, readErr := os.ReadDir(inputs[0])
			if readErr != nil {
				return "", nil, fmt.Errorf("%s: %w", inputs[0], readErr)
			}
			var names []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				ext := filepath.Ext(e.Name())
				if len(ext) > 0 {
					ext = ext[1:]
				}
				if messageFileExts[ext] {
					names = append(names, e.Name())
				}
			}
			sort.Strings(names)
			if len(names) == 0 {
				return "", nil, fmt.Errorf("%s: no .json, .toml, .yaml or .yml message files found", inputs[0])
			}
			return inputs[0], names, nil
		}
	}
	files = append(files, inputs...)
	sort.Strings(files)
	return "", files, nil
}

// loadFiles parses every input file with go-i18n's own message-file
// decoder, so a file msgparity accepts is a file go-i18n accepts, and a
// language tag msgparity assigns a file is the same tag go-i18n's Bundle
// would assign it.
func loadFiles(dir string, names []string) ([]loadedFile, []Finding) {
	var loaded []loadedFile
	var findings []Finding

	for _, name := range names {
		fullPath := name
		if dir != "" {
			fullPath = filepath.Join(dir, name)
		}
		buf, err := os.ReadFile(fullPath)
		if err != nil {
			findings = append(findings, Finding{
				Rule:     "load-error",
				Severity: SeverityError,
				File:     name,
				Message:  err.Error(),
			})
			continue
		}
		mf, err := i18n.ParseMessageFileBytes(buf, name, unmarshalFuncs)
		if err != nil {
			findings = append(findings, Finding{
				Rule:     "load-error",
				Severity: SeverityError,
				File:     name,
				Message:  fmt.Sprintf("could not parse: %s", err),
			})
			continue
		}
		if mf.Tag == language.Und {
			findings = append(findings, Finding{
				Rule:     "load-error",
				Severity: SeverityError,
				File:     name,
				Message:  "could not determine a locale from the filename (expected <name>.<lang>.<ext>)",
			})
			continue
		}
		loaded = append(loaded, loadedFile{Path: name, Locale: mf.Tag, Messages: mf.Messages})
	}

	return loaded, findings
}
