package cli

import "testing"

func TestPending(t *testing.T) {
	cli := NewCLI()

	cli.TestCommand("pending")
}
