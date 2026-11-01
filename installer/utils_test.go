package main

import "testing"

func TestExecCommandWithInputFeedsStdin(t *testing.T) {
	output, err := execCommandWithInput([]byte("hello"), "cat", 0)
	if err != nil {
		t.Fatalf("execCommandWithInput() returned error: %v", err)
	}
	if string(output) != "hello" {
		t.Errorf("output = %q, want %q", output, "hello")
	}
}

func TestExecCommandWithNilInputReadsEOF(t *testing.T) {
	output, err := execCommandWithInput(nil, "cat", 0)
	if err != nil {
		t.Fatalf("execCommandWithInput() returned error: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("output = %q, want empty", output)
	}
}

func TestExecCommandWithInputPropagatesExitError(t *testing.T) {
	if _, err := execCommandWithInput(nil, "exit 3", 0); err == nil {
		t.Fatal("execCommandWithInput() returned nil error for a failing command")
	}
}
