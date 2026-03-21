package campaigncontext

import (
	"strings"
)

// MemorySectionRead returns the trimmed body of the first ## heading that
// matches (case-insensitive) in content. If no match is found it returns
// ("", false).
func MemorySectionRead(content, heading string) (string, bool) {
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return "", false
	}
	for _, sec := range parseH2Sections(content) {
		if strings.EqualFold(sec.heading, heading) {
			return strings.TrimSpace(sec.body), true
		}
	}
	return "", false
}

// MemorySectionUpdate replaces the body of the matched ## heading in-place,
// or appends a new section at the end when no match exists. It returns the
// full updated document.
func MemorySectionUpdate(content, heading, body string) string {
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return content
	}

	sections := parseH2Sections(content)

	// Find existing section by case-insensitive match.
	found := false
	for i, sec := range sections {
		if strings.EqualFold(sec.heading, heading) {
			sections[i].body = body
			found = true
			break
		}
	}
	if !found {
		sections = append(sections, h2Section{heading: heading, body: body})
	}

	return renderH2Sections(sections)
}

// h2Section represents one ## heading and its body text.
type h2Section struct {
	heading string // empty heading means preamble (content before first ##)
	body    string
}

// parseH2Sections splits a markdown document into preamble + H2 sections.
// Each section starts at a `## ` line and extends to the next `## ` line or
// EOF.
func parseH2Sections(content string) []h2Section {
	lines := strings.Split(content, "\n")
	var sections []h2Section
	current := h2Section{} // preamble (empty heading)
	var bodyLines []string

	for _, line := range lines {
		if heading, ok := parseH2Line(line); ok {
			// Flush previous section.
			current.body = strings.Join(bodyLines, "\n")
			sections = append(sections, current)
			current = h2Section{heading: heading}
			bodyLines = nil
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	// Flush final section.
	current.body = strings.Join(bodyLines, "\n")
	sections = append(sections, current)
	return sections
}

// renderH2Sections reconstructs a document from parsed sections. It preserves
// the preamble and inserts a blank line between each heading line and its
// body, matching idiomatic markdown.
func renderH2Sections(sections []h2Section) string {
	var b strings.Builder
	for i, sec := range sections {
		if sec.heading == "" {
			// Preamble — write body as-is.
			text := strings.TrimRight(sec.body, "\n")
			b.WriteString(text)
			if text != "" {
				b.WriteString("\n")
			}
			continue
		}
		// Ensure a blank line before each heading (unless at document start).
		if i > 0 && b.Len() > 0 {
			// Ensure exactly one trailing newline, then add a blank line.
			cur := b.String()
			if !strings.HasSuffix(cur, "\n") {
				b.WriteString("\n")
			}
			if !strings.HasSuffix(cur, "\n\n") {
				b.WriteString("\n")
			}
		}
		b.WriteString("## ")
		b.WriteString(sec.heading)
		b.WriteString("\n")
		body := strings.Trim(sec.body, "\n")
		if body != "" {
			b.WriteString("\n")
			b.WriteString(body)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// parseH2Line checks whether line is an H2 markdown heading and returns the
// heading text.
func parseH2Line(line string) (string, bool) {
	trimmed := strings.TrimRight(line, " \t")
	if !strings.HasPrefix(trimmed, "## ") {
		return "", false
	}
	heading := strings.TrimSpace(trimmed[3:])
	if heading == "" {
		return "", false
	}
	return heading, true
}
