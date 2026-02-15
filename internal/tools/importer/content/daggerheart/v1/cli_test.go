package catalogimporter

import (
	"flag"
	"io"
	"testing"
)

func TestParseConfigRequiresDir(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	_, err := ParseConfig(fs, []string{})
	if err == nil {
		t.Fatal("expected error when dir is missing")
	}
}
