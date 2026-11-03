package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type cliOptions struct {
	kubeconfig  string
	showVersion bool
}

// parseOptions parses the command line in isolation from the global flag set so
// tests can exercise it repeatedly.
func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions

	fs := flag.NewFlagSet("om-kits-installer", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // main owns all user-facing output

	fs.BoolVar(&opts.showVersion, "version", false, "print version information and exit")
	fs.BoolVar(&opts.showVersion, "v", false, "print version information and exit (shorthand)")
	fs.StringVar(&opts.kubeconfig, "kubeconfig", "", "path to a single kubeconfig file")
	fs.StringVar(&opts.kubeconfig, "k", "", "path to a single kubeconfig file (shorthand)")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err // includes flag.ErrHelp for -h/--help
	}
	if rest := fs.Args(); len(rest) > 0 {
		return cliOptions{}, fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}

	return opts, nil
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: om-kits-installer [options]

Options:
  -v, --version          Print version information and exit.
  -k, --kubeconfig PATH  Use PATH as the kubeconfig for kubectl/helm child processes.
                         Precedence: this flag, then $KUBECONFIG, then
                         ~/.kube/config.
  -h, --help             Show this help message.
`)
}

// applyKubeconfig makes an explicit --kubeconfig win over the environment. It
// only mutates KUBECONFIG when the flag was supplied, preserving the existing
// env/default behavior otherwise.
func applyKubeconfig(flagValue string) error {
	if flagValue == "" {
		return nil
	}
	if strings.ContainsRune(flagValue, os.PathListSeparator) {
		return fmt.Errorf("--kubeconfig accepts a single file; use KUBECONFIG for a path list")
	}

	// Install tasks run with cmd.Dir set to the installer's directory, so a
	// relative path would resolve differently in child kubectl/helm processes.
	abs, err := filepath.Abs(flagValue)
	if err != nil {
		return fmt.Errorf("resolve kubeconfig %q: %w", flagValue, err)
	}
	if err := os.Setenv("KUBECONFIG", abs); err != nil {
		return fmt.Errorf("set KUBECONFIG to %q: %w", abs, err)
	}

	return nil
}
