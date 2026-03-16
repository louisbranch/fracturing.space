package render

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCreationCatalogInlineMarkdownRendersSupportedMarkers(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := creationCatalogInlineMarkdown(`Use **Hope** to become *swift*, then __strike__ a _Vulnerable_ foe.`).
		Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationCatalogInlineMarkdown: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`<strong class="font-semibold">Hope</strong>`,
		`<em class="italic">swift</em>`,
		`<strong class="font-semibold">strike</strong>`,
		`<em class="italic">Vulnerable</em>`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("creationCatalogInlineMarkdown output missing marker %q: %q", marker, got)
		}
	}
}

func TestCreationCatalogInlineMarkdownEscapesHTML(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := creationCatalogInlineMarkdown(`**Bold** <script>alert("x")</script> _safe_`).
		Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationCatalogInlineMarkdown: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, `<script>alert("x")</script>`) {
		t.Fatalf("creationCatalogInlineMarkdown output should escape raw html: %q", got)
	}
	for _, marker := range []string{
		`<strong class="font-semibold">Bold</strong>`,
		`&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`,
		`<em class="italic">safe</em>`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("creationCatalogInlineMarkdown output missing marker %q: %q", marker, got)
		}
	}
}

func TestCreationCatalogInlineMarkdownLeavesMalformedMarkersLiteral(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := creationCatalogInlineMarkdown(`Keep **broken and *odd markers literal.`).
		Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationCatalogInlineMarkdown: %v", err)
	}

	got := buf.String()
	if got != `Keep **broken and *odd markers literal.` {
		t.Fatalf("creationCatalogInlineMarkdown malformed output = %q", got)
	}
}
