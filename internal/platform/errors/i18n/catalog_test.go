package i18n

import "testing"

func TestGetCatalogFallback(t *testing.T) {
	base := GetCatalog("en-US")
	if base == nil {
		t.Fatal("expected base catalog")
	}
	fallback := GetCatalog("missing-locale")
	if fallback != base {
		t.Fatal("expected fallback to en-US catalog")
	}
}

func TestFormatFallbacks(t *testing.T) {
	cat := NewCatalog("test", map[Code]string{
		"code": "hello {{.Name}}",
	})

	if cat.Format("unknown", nil) != "unknown" {
		t.Fatal("expected code fallback when template missing")
	}
	if cat.Format("code", nil) != "hello <no value>" {
		t.Fatal("expected template to render missing metadata")
	}
}

func TestFormatTemplateErrorFallback(t *testing.T) {
	cat := NewCatalog("test", map[Code]string{
		"code": "{{ if .Name }}",
	})
	if cat.Format("code", map[string]string{"Name": "X"}) != "{{ if .Name }}" {
		t.Fatal("expected template fallback on parse error")
	}
}

func TestFormatTemplateExecutionErrorFallback(t *testing.T) {
	cat := NewCatalog("test", map[Code]string{
		"code": "{{ call .Name }}",
	})
	if cat.Format("code", map[string]string{"Name": "X"}) != "{{ call .Name }}" {
		t.Fatal("expected template fallback on execute error")
	}
}

func TestRegisterCatalog(t *testing.T) {
	custom := NewCatalog("custom", map[Code]string{"code": "ok"})
	RegisterCatalog("custom", custom)
	if got := GetCatalog("custom"); got != custom {
		t.Fatal("expected registered catalog")
	}
}
