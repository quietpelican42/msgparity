package msgparity

import (
	"sort"
	"text/template/parse"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// pluralCategories lists the CLDR category names in a fixed, CLDR-defined
// order: least to most specific, "other" last as the universal fallback.
var pluralCategories = []string{"zero", "one", "two", "few", "many", "other"}

// messageForms returns every non-empty (category, text) pair defined on a
// message, keyed by CLDR category name.
func messageForms(m *i18n.Message) map[string]string {
	forms := map[string]string{}
	for cat, text := range map[string]string{
		"zero":  m.Zero,
		"one":   m.One,
		"two":   m.Two,
		"few":   m.Few,
		"many":  m.Many,
		"other": m.Other,
	} {
		if text != "" {
			forms[cat] = text
		}
	}
	return forms
}

// builtinFuncNames are the function names text/template registers globally
// (https://pkg.go.dev/text/template#hdr-Functions) when a message is
// executed through the full template.Template, which go-i18n uses. parse.Parse
// on its own knows none of them, so without registering their names here
// every legitimate "{{printf ...}}" or "{{if eq .A .B}}" would be reported
// as a template-syntax error. A message that calls a function an
// application registers itself via a custom FuncMap is not something
// msgparity can know about statically; see the README's Limitations
// section.
var builtinFuncNames = []string{
	"and", "call", "html", "index", "slice", "js", "len", "not", "or",
	"print", "printf", "println", "urlquery",
	"eq", "ge", "gt", "le", "lt", "ne",
}

var builtinFuncs = func() map[string]interface{} {
	m := make(map[string]interface{}, len(builtinFuncNames))
	for _, name := range builtinFuncNames {
		m[name] = func() {}
	}
	return m
}()

// templateFields parses text using Go's text/template parser with the given
// delimiters (empty means the default "{{"/"}}") and returns the set of
// top-level field names referenced — the Ident of every FieldNode whose
// first identifier is a lookup on the template's root data, e.g. "Name" for
// both "{{.Name}}" and "{{.Name.First}}". A field referenced only through a
// rebound "." inside "{{range ...}}" or "{{with ...}}" is not top-level
// data and is not reported (see walkNode). It does not use regexp: a
// regexp over "{{...}}" cannot tell a field reference from a function
// call, a string literal, or a comment, and would either miss or
// over-report.
//
// An error is returned if text is not a valid Go template body; the caller
// treats that as a template-syntax finding rather than guessing at fields.
func templateFields(text, leftDelim, rightDelim string) (map[string]bool, error) {
	if leftDelim == "" {
		leftDelim = "{{"
	}
	if rightDelim == "" {
		rightDelim = "}}"
	}
	trees, err := parse.Parse("msgparity", text, leftDelim, rightDelim, builtinFuncs)
	if err != nil {
		return nil, err
	}
	fields := map[string]bool{}
	for _, tree := range trees {
		if tree == nil || tree.Root == nil {
			continue
		}
		walkNode(tree.Root, fields, true)
	}
	return fields, nil
}

// walkNode walks a template's parse tree collecting field names. topLevel
// is false inside a "{{range ...}}" or "{{with ...}}" body, where "." (and
// so any bare ".Field" reference) is rebound to the loop element or the
// with's pipe value rather than the template's root data — a "{{.Bio}}"
// inside "{{with .Profile}}...{{end}}" refers to Profile.Bio, not a
// top-level "Bio" field, so it is not collected. The controlling pipe of a
// range/with, and its ElseList (which runs in the outer, unrebound scope
// when the range is empty or the with's value is falsy), are walked at the
// caller's topLevel value, not false.
func walkNode(n parse.Node, fields map[string]bool, topLevel bool) {
	if n == nil {
		return
	}
	switch node := n.(type) {
	case *parse.FieldNode:
		if topLevel && len(node.Ident) > 0 {
			fields[node.Ident[0]] = true
		}
	case *parse.ChainNode:
		walkNode(node.Node, fields, topLevel)
	case *parse.PipeNode:
		if node == nil {
			return
		}
		for _, cmd := range node.Cmds {
			walkNode(cmd, fields, topLevel)
		}
		for _, decl := range node.Decl {
			walkNode(decl, fields, topLevel)
		}
	case *parse.CommandNode:
		for _, arg := range node.Args {
			walkNode(arg, fields, topLevel)
		}
	case *parse.ActionNode:
		walkNode(node.Pipe, fields, topLevel)
	case *parse.IfNode:
		// "if" does not rebind ".".
		walkNode(node.Pipe, fields, topLevel)
		walkNode(node.List, fields, topLevel)
		walkNode(node.ElseList, fields, topLevel)
	case *parse.RangeNode:
		walkNode(node.Pipe, fields, topLevel)
		walkNode(node.List, fields, false)
		walkNode(node.ElseList, fields, topLevel)
	case *parse.WithNode:
		walkNode(node.Pipe, fields, topLevel)
		walkNode(node.List, fields, false)
		walkNode(node.ElseList, fields, topLevel)
	case *parse.TemplateNode:
		walkNode(node.Pipe, fields, topLevel)
	case *parse.ListNode:
		if node == nil {
			return
		}
		for _, c := range node.Nodes {
			walkNode(c, fields, topLevel)
		}
	case *parse.VariableNode:
		// A local variable ($x), not a field on TemplateData. Its own
		// definition (the right-hand side of "$x := .Name") is reached via
		// the enclosing PipeNode's Decl, so nothing is lost by ignoring the
		// variable reference itself.
	}
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TemplateFields returns the sorted, deduplicated set of top-level field
// names a Go template body references, e.g. []string{"Count", "Name"} for
// "{{.Name}} has {{.Count}} cats.". It is exported so callers can reuse
// msgparity's field extraction (parsed with text/template/parse, not
// regexp) outside of a full Check run — for example, to lint a message
// file's messages against a Go struct's fields.
func TemplateFields(text, leftDelim, rightDelim string) ([]string, error) {
	fields, err := templateFields(text, leftDelim, rightDelim)
	if err != nil {
		return nil, err
	}
	return sortedKeys(fields), nil
}
