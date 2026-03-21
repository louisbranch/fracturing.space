package memorydoc

import "strings"

// SectionRead returns the trimmed body of the first ## heading that matches
// heading case-insensitively. If no match is found it returns ("", false).
func SectionRead(content, heading string) (string, bool) {
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

// SectionUpdate replaces the body of the matched ## heading in-place, or
// appends a new section at the end when no match exists. It returns the full
// updated document.
func SectionUpdate(content, heading, body string) string {
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return content
	}

	sections := parseH2Sections(content)
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

type h2Section struct {
	heading string
	body    string
}

func parseH2Sections(content string) []h2Section {
	lines := strings.Split(content, "\n")
	var sections []h2Section
	current := h2Section{}
	var bodyLines []string

	for _, line := range lines {
		if heading, ok := parseH2Line(line); ok {
			current.body = strings.Join(bodyLines, "\n")
			sections = append(sections, current)
			current = h2Section{heading: heading}
			bodyLines = nil
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	current.body = strings.Join(bodyLines, "\n")
	sections = append(sections, current)
	return sections
}

func renderH2Sections(sections []h2Section) string {
	var b strings.Builder
	for i, sec := range sections {
		if sec.heading == "" {
			text := strings.TrimRight(sec.body, "\n")
			b.WriteString(text)
			if text != "" {
				b.WriteString("\n")
			}
			continue
		}
		if i > 0 && b.Len() > 0 {
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
