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

func TestParseConfigParsesSkipIfReady(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg, err := ParseConfig(fs, []string{"-dir", ".", "-skip-if-ready"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.SkipIfReady {
		t.Fatal("SkipIfReady = false, want true")
	}
}
