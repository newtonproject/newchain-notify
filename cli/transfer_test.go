package cli

import "testing"

func TestTransfer(t *testing.T) {
	cli := NewCLI()

	cli.TestCommand("transfer")
}
