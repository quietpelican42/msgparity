# msgparity

A linter for go-i18n message files that catches a translation dropping a
template placeholder or missing a plural form the language requires.

```sh
msgparity check --source en locales/
```

```
es.toml: [es] Greeting: placeholder-missing: error: translation drops 2 placeholders used by the source message: {{.Count}}, {{.Name}}
pl.toml: [pl] PersonCats: plural-missing: error: missing plural category "few", which pl's CLDR cardinal rules require for this message
pl.toml: [pl] PersonCats: plural-missing: error: missing plural category "many", which pl's CLDR cardinal rules require for this message
pl.toml: [pl] PersonCats: plural-missing: error: missing plural category "one", which pl's CLDR cardinal rules require for this message
msgparity: 4 error(s), 0 warning(s)
```

## Why

[go-i18n](https://github.com/nicksnyder/go-i18n) (v2) loads translations from
JSON, TOML or YAML message files and interpolates them with Go's
`text/template`. Two mistakes in those files compile fine, pass
`goi18n merge`, and only surface when go-i18n actually renders the affected
message with the right plural count:

- A translation drops a placeholder the source message uses (a whole field,
  or a typo of one).
- A translation is missing a CLDR plural category its own language requires
  — Polish needs `one`/`few`/`many`/`other`; English only needs `one`/`other`.
  A translator who only fills in `other` (the one every language accepts) has
  silently broken every other count.

`goi18n merge` only tracks which message IDs have a translation; it never
looks at what a translation's template actions or plural forms are.

Measured: `goi18n merge en.toml es.toml`, where `es.toml`'s only translated
string drops both placeholders the source message uses:

```
goi18n exit code: 0

# active.es.toml
[Greeting]
hash = "sha1-9b2655725c306ed42028f0082b997f8f0a2bba70"
other = "Hola!"
```

No warning, exit 0, and the file it just wrote is missing the two fields
`en.toml`'s `Greeting` interpolates. At runtime go-i18n does not error either
— `Localize` just renders `"Hola!"` and silently drops the caller's data:

```
msg="Hola!" err=<nil>
```

The plural case is worse because go-i18n *does* return an error — but only
for counts the incomplete language happens to hit, and only if the caller
checks the error:

```
pl count=  0: msg="Ana ma 0 kotów." err=message "PersonCats" has no plural form "many"
pl count=  1: msg="Ana ma 1 kotów." err=message "PersonCats" has no plural form "one"
pl count=  2: msg="Ana ma 2 kotów." err=message "PersonCats" has no plural form "few"
```

`msg` is non-empty in every case (go-i18n falls back to the `other` text), so
a caller that logs but doesn't surface the error ships grammatically wrong
Polish text with no visible symptom.

A search of pkg.go.dev and GitHub for "go-i18n" together with "lint",
"plural", and "placeholder" turned up no existing go-i18n-aware linter.
[Lisan](https://github.com/persona-id/lisan), a newer i18n library, validates
only its own message format and does not read go-i18n files.

## Installation

```sh
go install github.com/__GH_OWNER__/msgparity/cmd/msgparity@latest
```

Or as a library:

```sh
go get github.com/__GH_OWNER__/msgparity
```

## Usage

### Check a directory of message files

go-i18n message files are flat: one file per locale, named
`<name>.<lang>.<ext>` or `<lang>.<ext>` (`active.es.toml`, `es.toml`).
`msgparity check` reads every `.json`, `.toml` and `.yaml`/`.yml` file
directly inside a directory:

```sh
msgparity check locales/
```

### Check an explicit file list

```sh
msgparity check locales/active.en.toml locales/active.es.toml
```

### Use a source locale other than English

```sh
msgparity check --source fr locales/
```

### Wire into CI

```sh
msgparity check --format github locales/ || exit 1
```

`--format github` prints [workflow command
annotations](https://docs.github.com/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message)
that GitHub Actions turns into inline PR review comments.

### Use the field extractor as a library

```go
fields, err := msgparity.TemplateFields("{{.Name}} has {{.Count}} cats.", "", "")
// fields == []string{"Count", "Name"}
```

## Reference

### CLI

```
msgparity check [--source <locale>] [--format text|json|github] <dir|files...>
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--source` | `en` | Locale every other locale's messages are checked against. |
| `--format` | `text` | `text`, `json`, or `github`. |

Exit codes: `0` no findings, `1` findings (error or warning severity), `2`
could not load or parse the input (bad flags, missing directory, no message
file for the source locale, an unparsable file).

### Rules

| Rule | Severity | Fires when |
| --- | --- | --- |
| `placeholder-missing` | error | A translation's plural forms, together, reference fewer template fields than the source message. |
| `placeholder-extra` | error | A translation references a template field the source message does not use (usually a typo). |
| `template-syntax` | error | A plural form is not a valid Go template. |
| `plural-missing` | error | A translation lacks a CLDR plural category its language requires, for a message the source marks plural (more than one non-empty form). |
| `plural-unused` | warning | A translation defines a plural category its language's CLDR rules never select — dead text, not a runtime bug. |
| `missing-id` | warning | A source message has no entry at all in another locale's files. |
| `orphan-id` | warning | A locale defines a message ID the source locale does not have. |
| `duplicate-id` | error | The same message ID is defined in more than one file for one locale. |

### Go API

`Check(input []string, opts Options) (Result, error)` runs every rule.
`WriteResult(w io.Writer, result Result, format Format) error` formats a
`Result`. `TemplateFields(text, leftDelim, rightDelim string) ([]string,
error)` exposes the field extractor on its own.

## How it works

### Parsing

msgparity calls go-i18n's own `i18n.ParseMessageFileBytes` (a public
function, not something reimplemented) with go-i18n's own
filename-to-locale rule, and its JSON decoder plus `BurntSushi/toml` and
`gopkg.in/yaml.v3` registered as go-i18n `UnmarshalFunc`s — the same
unmarshalers a real project registers on its `i18n.Bundle`. A file msgparity
loads is a file go-i18n loads, with the same locale tag and the same
message-ID-to-plural-form structure; msgparity never has its own opinion
about message-file syntax that could disagree with go-i18n's. That is also
why `github.com/nicksnyder/go-i18n/v2` is a direct dependency rather than
just a test fixture source.

### Placeholders

Each plural form's text is parsed with `text/template/parse.Parse` — the
same parser `text/template` itself uses — and every `FieldNode` reachable
from the root data context is collected as a used field. This is why it is
not a regexp over `{{...}}`: a regexp cannot tell a field reference from a
function call, a string literal or a `range` loop variable, and doing that
wrong produces exactly the false positives/negatives a linter should not
have. A field referenced only through a rebound `.` inside `{{range}}` or
`{{with}}` is correctly not counted as top-level data (see `walkNode` in
`placeholders.go`). The standard `text/template` builtin function names
(`printf`, `index`, `eq`, ...) are registered so legitimate messages that use
them parse; a custom function an application registers itself is invisible
to msgparity (see Limitations).

### Plural categories

`golang.org/x/text/feature/plural`'s `plural.Cardinal.MatchPlural` is
sampled over every integer 0-199 (CLDR cardinal rules are defined in terms
of a number's last one or two decimal digits, so this range covers every
modulus pattern actually used) plus a few larger round numbers, and the
union of categories returned is the set that language "requires." go-i18n's
own plural-rule table (generated from CLDR the same way) is unexported
(`internal/plural`), so it cannot be imported; x/text's CLDR-generated
cardinal rules are the closest public, independently-authoritative source.
This was cross-checked against go-i18n's real runtime behavior for the
languages the build brief named — Polish (one/few/many/other), Arabic (all
six categories) and Japanese (other only) — see `TestRequiredCategories`.

## Limitations

- The "many" exception for exact millions in French, and any other
  large-number-dependent CLDR rule, is not detected. Measured: go-i18n
  requires a `many` form for French when `PluralCount` is an exact multiple
  of 1,000,000 (verified against `nicksnyder/go-i18n` v2.6.1 at
  `Count=1000000`, `2000000`, `21000000`), but
  `golang.org/x/text/feature/plural` v0.42.0's sampled cardinal rules for
  `fr` never select `many` for the integers msgparity samples. This is a gap
  between x/text's public CLDR rule tables and go-i18n's own, not something
  msgparity can currently close without vendoring CLDR data directly.
- A custom template function an application registers via its own `FuncMap`
  at `Execute` time is invisible to msgparity. A message calling such a
  function is reported as `template-syntax`, even though go-i18n would
  render it correctly at runtime. The standard library's builtin function
  names are registered so ordinary messages are unaffected.
- `plural-missing`/`plural-unused` only apply to messages the source marks
  plural (more than one non-empty form). If application code attaches a
  `PluralCount` to a message the source only ever defines as `other`,
  go-i18n still runs CLDR category resolution for that count and can error
  for languages/counts that don't resolve to `other` — msgparity does not
  flag this, because the message was never written as a plural message.
- A field referenced through `$.Field` from inside a rebound `{{with}}` or
  `{{range}}` body is not detected as a top-level field. This is rare in
  translator-facing strings; ordinary `{{.Field}}` references are unaffected
  at any nesting depth `if` introduces (`if` does not rebind `.`).
- Locale matching for `--source` is an exact BCP 47 tag match against the
  tag go-i18n's own filename parser derives (`en.toml` → `en`). `en-US.toml`
  is not matched by `--source en`.
- No auto-fix. msgparity only reports.

## Compatibility

Built and tested against `github.com/nicksnyder/go-i18n/v2` v2.6.1 and
`golang.org/x/text` v0.42.0, on Go 1.27.1 (darwin/arm64). CI runs the same
checks on Go 1.27 on Ubuntu; that combination is run in CI, not separately
verified here.

## Development

```sh
go build ./...
go vet ./...
go test -race -count=1 ./...
gofmt -l .
```

## License

MIT. See [LICENSE](LICENSE).
