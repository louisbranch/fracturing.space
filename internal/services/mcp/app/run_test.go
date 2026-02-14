package mcp

import (
	"context"
	"testing"
)

func TestRunRejectsInvalidTransport(t *testing.T) {
	err := Run(context.Background(), "", "", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
}
