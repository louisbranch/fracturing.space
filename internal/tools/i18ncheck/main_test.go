package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	repoRoot := writeFixtureRepo(t, map[string]string{
		"internal/services/worker/domain/notifications.go": `package domain

import (
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	notificationpayload "github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
)

func build() notificationpayload.InAppPayload {
	return notificationpayload.InAppPayload{
		Title: platformi18n.NewCopyRef("notification.campaign_invite.created.title"),
		Body:  platformi18n.NewCopyRef("notification.campaign_invite.created.body_summary", "gm", "Skyfall"),
	}
}
`,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-repo-root", repoRoot}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "i18n catalog check passed") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}
}

func TestRunMissingBaseLocale(t *testing.T) {
	repoRoot := writeFixtureRepo(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-base-locale", "zz-ZZ", "-repo-root", repoRoot}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := exitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `base locale "zz-ZZ" is missing from catalogs`) {
		t.Fatalf("expected missing locale error, got %q", stderr.String())
	}
}

func TestCollectSourceFailuresFlagsConvertedSeams(t *testing.T) {
	repoRoot := writeFixtureRepo(t, map[string]string{
		"internal/services/worker/domain/notifications.go": `package domain

import (
	"fmt"

	notificationpayload "github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
)

func build(recipient string) notificationpayload.InAppPayload {
	payload := notificationpayload.InAppPayload{}
	payload.Title = "Invite for " + recipient
	payload.Body = fmt.Sprintf("%s invited you", recipient)
	return notificationpayload.InAppPayload{
		Title: "raw notification title",
		Facts: []notificationpayload.PayloadFact{{
			Label: "Campaign",
			Value: recipient,
		}},
		Actions: []notificationpayload.PayloadAction{{
			Label: "Open invite",
			Kind:  "open",
		}},
		Body: payload.Body,
	}
}
`,
		"internal/services/web/modules/example/flash.go": `package example

import (
	"net/http"

	flash "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

func writeNotice(w http.ResponseWriter, r *http.Request) {
	flash.Write(w, r, flash.Notice{
		Kind:    flash.KindError,
		Message: "Profile updated",
		Key:     "Profile updated",
	})
	var notice flash.Notice
	notice.Key = "Invite sent"
	flash.Write(w, r, flash.NoticeSuccess("Saved settings"))
}
`,
		"internal/services/web/platform/httpx/httpx.go": `package httpx

import "net/http"

func writeErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
`,
	})

	failures, err := collectSourceFailures(repoRoot)
	if err != nil {
		t.Fatalf("collectSourceFailures returned error: %v", err)
	}
	got := strings.Join(failures, "\n")

	wantContains := []string{
		"internal/services/worker/domain/notifications.go:",
		"raw notification payload Title",
		"raw notification payload Body",
		"raw notification payload Label",
		"internal/services/web/modules/example/flash.go:",
		"raw flash notice Message",
		"raw flash notice Key",
		"raw flash notice key passed to NoticeSuccess",
		"internal/services/web/platform/httpx/httpx.go:",
		"request-facing err.Error() fallback passed to http.Error",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected failure %q in output:\n%s", want, got)
		}
	}
}

func TestCollectSourceFailuresAllowsKeysAndSuppression(t *testing.T) {
	repoRoot := writeFixtureRepo(t, map[string]string{
		"internal/services/worker/domain/notifications.go": `package domain

import (
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	notificationpayload "github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
)

func build(owner string) notificationpayload.InAppPayload {
	payload := notificationpayload.InAppPayload{
		Title: platformi18n.NewCopyRef("notification.campaign_invite.created.title"),
		Body: platformi18n.CopyRef{
			Key:  "notification.campaign_invite.created.body_summary",
			Args: []string{owner, "Skyfall"},
		},
	}
	payload.Title = "OpenAI" // i18n:allow-raw upstream brand token required by external contract
	return payload
}
`,
		"internal/services/web/modules/example/flash.go": `package example

import (
	"net/http"

	flash "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

func writeNotice(w http.ResponseWriter, r *http.Request, err error) {
	flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: "error.web.message.failed_to_create_invite"})
	// i18n:allow-raw legacy endpoint still awaiting localization
	http.Error(w, err.Error(), http.StatusBadRequest)
}
`,
	})

	failures, err := collectSourceFailures(repoRoot)
	if err != nil {
		t.Fatalf("collectSourceFailures returned error: %v", err)
	}
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCollectSourceFailuresRequiresSuppressionReason(t *testing.T) {
	repoRoot := writeFixtureRepo(t, map[string]string{
		"internal/services/web/modules/example/flash.go": `package example

import (
	"net/http"

	flash "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

func writeNotice(w http.ResponseWriter, r *http.Request) {
	flash.Write(w, r, flash.Notice{
		Kind: flash.KindError,
		// i18n:allow-raw
		Key: "Profile updated",
	})
}
`,
	})

	failures, err := collectSourceFailures(repoRoot)
	if err != nil {
		t.Fatalf("collectSourceFailures returned error: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected one failure, got %v", failures)
	}
	if !strings.Contains(failures[0], "raw flash notice Key") {
		t.Fatalf("expected raw flash key failure, got %v", failures)
	}
}

func TestPrintfAndTemplateTokens(t *testing.T) {
	gotPrintf := printfTokens("Hello %[2]s from %s and %d%% done")
	wantPrintf := []string{"%[2]s", "%d", "%s"}
	if !reflect.DeepEqual(gotPrintf, wantPrintf) {
		t.Fatalf("printfTokens() = %#v, want %#v", gotPrintf, wantPrintf)
	}

	gotTemplate := templateTokens("Hi {{ .Name }} from {{.Place}}")
	wantTemplate := []string{"Name", "Place"}
	if !reflect.DeepEqual(gotTemplate, wantTemplate) {
		t.Fatalf("templateTokens() = %#v, want %#v", gotTemplate, wantTemplate)
	}
}

func TestLooksLikeLocalizationKeyLiteral(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "web.invite.state.claimed.heading", want: true},
		{value: "ERROR_WEB_MESSAGE_FAILED", want: true},
		{value: "Failed to create invite", want: false},
		{value: "OpenAI", want: false},
		{value: "profile updated", want: false},
	}

	for _, tc := range tests {
		if got := looksLikeLocalizationKeyLiteral(tc.value); got != tc.want {
			t.Fatalf("looksLikeLocalizationKeyLiteral(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestContainsErrorMethodCall(t *testing.T) {
	expr, err := parser.ParseExpr(`fmt.Sprintf("bad: %s", err.Error())`)
	if err != nil {
		t.Fatalf("ParseExpr() error = %v", err)
	}
	if !containsErrorMethodCall(expr) {
		t.Fatal("containsErrorMethodCall() = false, want true")
	}

	expr, err = parser.ParseExpr(`fmt.Sprintf("bad: %s", msg)`)
	if err != nil {
		t.Fatalf("ParseExpr() error = %v", err)
	}
	if containsErrorMethodCall(expr) {
		t.Fatal("containsErrorMethodCall() = true, want false")
	}
}

func TestLooksLikeRawNotificationExpr(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", `package fixture

import (
	"fmt"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

var (
	rawString = "Invite ready"
	keyString = "notification.campaign_invite.created.title"
	rawFormat = fmt.Sprintf("%s invited you", "gm")
	keyRef = platformi18n.NewCopyRef("notification.campaign_invite.created.body_summary", "gm", "Skyfall")
	rawRef = platformi18n.NewCopyRef("Invite ready")
)
`, 0)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	imports := fileImportMap(file)
	values := map[string]bool{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) == 0 || len(valueSpec.Values) == 0 {
				continue
			}
			values[valueSpec.Names[0].Name] = looksLikeRawNotificationExpr(valueSpec.Values[0], imports)
		}
	}

	want := map[string]bool{
		"rawString": true,
		"keyString": false,
		"rawFormat": true,
		"keyRef":    false,
		"rawRef":    true,
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("raw notification detection = %#v, want %#v", values, want)
	}
}

func writeFixtureRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, contents := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}
	return root
}
