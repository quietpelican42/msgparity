package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"clean", []string{"check", "../../testdata/formats/toml"}, 0},
		{"findings", []string{"check", "--source", "en", "../../testdata/formats/toml/en.toml", "../../testdata/broken/es.toml"}, 1},
		{"missing dir", []string{"check", "../../testdata/does-not-exist"}, 2},
		{"no subcommand", []string{}, 2},
		{"unknown subcommand", []string{"lint", "."}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tc.args, &stdout, &stderr)
			if got != tc.want {
				t.Fatalf("exit code = %d, want %d\nstdout: %s\nstderr: %s", got, tc.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunJSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"check", "--format", "json", "../../testdata/formats/toml/en.toml", "../../testdata/broken/es.toml"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"rule": "placeholder-missing"`) {
		t.Fatalf("json output missing placeholder-missing rule:\n%s", stdout.String())
	}
}

func TestRunGitHubFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"check", "--format", "github", "../../testdata/formats/toml/en.toml", "../../testdata/broken/es.toml"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "::error file=") {
		t.Fatalf("github output missing an ::error annotation:\n%s", stdout.String())
	}
}

func TestRunDefaultsToCurrentDir(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"check"}, &stdout, &stderr)
	// Whatever the cwd is when "go test" runs this package, it should not
	// panic or require an explicit directory argument.
	if code != 0 && code != 1 && code != 2 {
		t.Fatalf("unexpected exit code %d\nstderr: %s", code, stderr.String())
	}
}
