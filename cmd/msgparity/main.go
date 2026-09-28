// Command msgparity checks go-i18n message files for translations that drop
// or mistype a template placeholder, or that are missing a CLDR plural
// category their language requires — neither of which goi18n merge catches,
// and both of which only fail at the moment go-i18n actually renders that
// message with that count.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/__GH_OWNER__/msgparity"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("msgparity", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sourceFlag := fs.String("source", "en", "locale every other locale's messages are checked against")
	formatFlag := fs.String("format", "text", "output format: text, json or github")
	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage: msgparity check [flags] <dir|files...>

Checks go-i18n message files for translations that drop or mistype a
template placeholder the source message uses, or that are missing a CLDR
plural category their language requires.

Flags:
`)
		fs.PrintDefaults()
	}

	if len(args) == 0 || args[0] != "check" {
		fs.Usage()
		return 2
	}
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	inputs := fs.Args()
	if len(inputs) == 0 {
		inputs = []string{"."}
	}

	result, err := msgparity.Check(inputs, msgparity.Options{Source: *sourceFlag})
	if err != nil {
		fmt.Fprintf(stderr, "msgparity: %s\n", err)
		return 2
	}

	if err := msgparity.WriteResult(stdout, result, msgparity.Format(*formatFlag)); err != nil {
		fmt.Fprintf(stderr, "msgparity: %s\n", err)
		return 2
	}

	if len(result.Findings) > 0 {
		return 1
	}
	return 0
}
