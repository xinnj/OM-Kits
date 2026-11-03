package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    cliOptions
		wantErr bool
	}{
		{name: "no args", args: nil, want: cliOptions{}},
		{name: "long version", args: []string{"--version"}, want: cliOptions{showVersion: true}},
		{name: "short version", args: []string{"-v"}, want: cliOptions{showVersion: true}},
		{name: "kubeconfig equals", args: []string{"--kubeconfig=/tmp/k"}, want: cliOptions{kubeconfig: "/tmp/k"}},
		{name: "kubeconfig space", args: []string{"--kubeconfig", "/tmp/k"}, want: cliOptions{kubeconfig: "/tmp/k"}},
		{name: "kubeconfig shorthand", args: []string{"-k", "/tmp/k"}, want: cliOptions{kubeconfig: "/tmp/k"}},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
		{name: "missing value", args: []string{"--kubeconfig"}, wantErr: true},
		{name: "positional argument", args: []string{"extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptions(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOptions(%v) = nil error, want error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions(%v) error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("parseOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseOptionsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			_, err := parseOptions([]string{arg})
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("parseOptions(%q) error = %v, want flag.ErrHelp", arg, err)
			}
		})
	}
}

func TestApplyKubeconfig(t *testing.T) {
	t.Run("empty flag preserves environment", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "/from/env")
		if err := applyKubeconfig(""); err != nil {
			t.Fatalf("applyKubeconfig(\"\") error: %v", err)
		}
		if got := os.Getenv("KUBECONFIG"); got != "/from/env" {
			t.Fatalf("KUBECONFIG = %q, want %q", got, "/from/env")
		}
	})

	t.Run("relative flag becomes absolute", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "/from/env")
		if err := applyKubeconfig("relative/kubeconfig"); err != nil {
			t.Fatalf("applyKubeconfig(\"relative/kubeconfig\") error: %v", err)
		}
		want, err := filepath.Abs("relative/kubeconfig")
		if err != nil {
			t.Fatalf("filepath.Abs: %v", err)
		}
		if got := os.Getenv("KUBECONFIG"); got != want {
			t.Fatalf("KUBECONFIG = %q, want %q", got, want)
		}
	})

	t.Run("path list is rejected", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "/from/env")
		value := "a" + string(os.PathListSeparator) + "b"
		if err := applyKubeconfig(value); err == nil {
			t.Fatalf("applyKubeconfig(%q) = nil error, want error", value)
		}
	})
}
