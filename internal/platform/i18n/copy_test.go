package i18n

import (
	"fmt"
	"testing"

	"golang.org/x/text/message"
)

func TestNormalizeCopyRef(t *testing.T) {
	t.Parallel()

	got, ok := NormalizeCopyRef(CopyRef{
		Key:  " notification.invite.title ",
		Args: []string{" owner ", " Campaign "},
	})
	if !ok {
		t.Fatal("NormalizeCopyRef() ok = false, want true")
	}
	if got.Key != "notification.invite.title" {
		t.Fatalf("key = %q, want %q", got.Key, "notification.invite.title")
	}
	if len(got.Args) != 2 || got.Args[0] != "owner" || got.Args[1] != "Campaign" {
		t.Fatalf("args = %#v, want trimmed args", got.Args)
	}
}

func TestNormalizeCopyRefRejectsBlankKey(t *testing.T) {
	t.Parallel()

	if _, ok := NormalizeCopyRef(CopyRef{Key: "   "}); ok {
		t.Fatal("NormalizeCopyRef() ok = true, want false")
	}
}

func TestResolveCopy(t *testing.T) {
	t.Parallel()

	loc := copyResolverStub{
		values: map[string]string{
			"notification.invite.body": "@%[1]s invited you to join %[2]s.",
		},
	}
	got := ResolveCopy(loc, NewCopyRef("notification.invite.body", "owner", "Skyfall"))
	if got != "@owner invited you to join Skyfall." {
		t.Fatalf("ResolveCopy() = %q, want formatted value", got)
	}
}

func TestResolveCopyWithoutLocalizerReturnsKey(t *testing.T) {
	t.Parallel()

	if got := ResolveCopy(nil, NewCopyRef("notification.invite.title")); got != "notification.invite.title" {
		t.Fatalf("ResolveCopy() = %q, want key fallback", got)
	}
}

type copyResolverStub struct {
	values map[string]string
}

func (s copyResolverStub) Sprintf(key message.Reference, args ...any) string {
	typed, _ := key.(string)
	template := s.values[typed]
	if template == "" {
		return typed
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}
