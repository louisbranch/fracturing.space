package render

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// creationCatalogInlineKind identifies which safe inline wrapper, if any,
// should surround a parsed catalog text segment at render time.
type creationCatalogInlineKind uint8

const (
	creationCatalogInlinePlain creationCatalogInlineKind = iota
	creationCatalogInlineStrong
	creationCatalogInlineEm
)

// creationCatalogInlineSegment keeps parse output explicit so templates can
// render only the tiny markup subset this view owns.
type creationCatalogInlineSegment struct {
	text string
	kind creationCatalogInlineKind
}

// creationCatalogInlineMarkdown renders limited inline emphasis for
// system-owned catalog copy while keeping all non-markup text HTML-escaped.
func creationCatalogInlineMarkdown(text string) templ.Component {
	segments := parseCreationCatalogInlineMarkdown(text)
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, segment := range segments {
			if err := writeCreationCatalogInlineSegment(w, segment); err != nil {
				return err
			}
		}
		return nil
	})
}

// writeCreationCatalogInlineSegment emits one parsed segment while ensuring
// every text payload still flows through templ escaping.
func writeCreationCatalogInlineSegment(w io.Writer, segment creationCatalogInlineSegment) error {
	switch segment.kind {
	case creationCatalogInlineStrong:
		if _, err := io.WriteString(w, `<strong class="font-semibold">`); err != nil {
			return err
		}
		if _, err := io.WriteString(w, templ.EscapeString(segment.text)); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</strong>`)
		return err
	case creationCatalogInlineEm:
		if _, err := io.WriteString(w, `<em class="italic">`); err != nil {
			return err
		}
		if _, err := io.WriteString(w, templ.EscapeString(segment.text)); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</em>`)
		return err
	default:
		_, err := io.WriteString(w, templ.EscapeString(segment.text))
		return err
	}
}

// parseCreationCatalogInlineMarkdown recognizes only inline emphasis markers so
// catalog copy can stay system-authored while the web layer controls HTML.
func parseCreationCatalogInlineMarkdown(text string) []creationCatalogInlineSegment {
	if text == "" {
		return nil
	}

	segments := make([]creationCatalogInlineSegment, 0, 4)
	textStart := 0

	for i := 0; i < len(text); {
		token, kind := creationCatalogInlineMarkerAt(text, i)
		if token == "" {
			i++
			continue
		}

		contentStart := i + len(token)
		contentEnd := strings.Index(text[contentStart:], token)
		if contentEnd < 0 {
			i += len(token)
			continue
		}
		contentEnd += contentStart
		if contentEnd == contentStart {
			i += len(token)
			continue
		}

		if textStart < i {
			segments = append(segments, creationCatalogInlineSegment{
				text: text[textStart:i],
				kind: creationCatalogInlinePlain,
			})
		}
		segments = append(segments, creationCatalogInlineSegment{
			text: text[contentStart:contentEnd],
			kind: kind,
		})

		i = contentEnd + len(token)
		textStart = i
	}

	if textStart < len(text) {
		segments = append(segments, creationCatalogInlineSegment{
			text: text[textStart:],
			kind: creationCatalogInlinePlain,
		})
	}

	if len(segments) == 0 {
		return []creationCatalogInlineSegment{{
			text: text,
			kind: creationCatalogInlinePlain,
		}}
	}

	return segments
}

// creationCatalogInlineMarkerAt centralizes token precedence so strong markers
// win over single-character emphasis when both could match at one offset.
func creationCatalogInlineMarkerAt(text string, index int) (string, creationCatalogInlineKind) {
	if index < 0 || index >= len(text) {
		return "", creationCatalogInlinePlain
	}

	switch {
	case strings.HasPrefix(text[index:], "**"):
		return "**", creationCatalogInlineStrong
	case strings.HasPrefix(text[index:], "__"):
		return "__", creationCatalogInlineStrong
	case strings.HasPrefix(text[index:], "*"):
		return "*", creationCatalogInlineEm
	case strings.HasPrefix(text[index:], "_"):
		return "_", creationCatalogInlineEm
	default:
		return "", creationCatalogInlinePlain
	}
}
