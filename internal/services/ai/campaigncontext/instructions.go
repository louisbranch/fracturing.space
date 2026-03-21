package campaigncontext

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	instructions "github.com/louisbranch/fracturing.space/data/instructions"
)

// InstructionLoader reads and composes instruction files from a root directory.
// It supports both embedded defaults and filesystem overrides for development.
type InstructionLoader struct {
	fs fs.FS
}

// NewInstructionLoader builds an InstructionLoader. If root is empty, the
// embedded default instruction set is used. Otherwise, the loader reads from
// the filesystem at root.
func NewInstructionLoader(root string) *InstructionLoader {
	root = strings.TrimSpace(root)
	if root == "" {
		return &InstructionLoader{fs: instructions.V1}
	}
	return &InstructionLoader{fs: os.DirFS(root)}
}

// LoadCoreSkills reads core/skills.md.
func (l *InstructionLoader) LoadCoreSkills() (string, error) {
	return l.readFile("v1/core/skills.md")
}

// LoadCoreInteraction reads core/interaction.md.
func (l *InstructionLoader) LoadCoreInteraction() (string, error) {
	return l.readFile("v1/core/interaction.md")
}

// LoadCoreMemoryGuide reads core/memory-guide.md.
func (l *InstructionLoader) LoadCoreMemoryGuide() (string, error) {
	return l.readFile("v1/core/memory-guide.md")
}

// LoadSystemSkills reads {system}/skills.md. Returns empty string (no error)
// if the file does not exist.
func (l *InstructionLoader) LoadSystemSkills(system string) (string, error) {
	return l.readOptionalFile(fmt.Sprintf("v1/%s/skills.md", system))
}

// LoadSystemReferenceGuide reads {system}/reference-guide.md. Returns empty
// string (no error) if the file does not exist.
func (l *InstructionLoader) LoadSystemReferenceGuide(system string) (string, error) {
	return l.readOptionalFile(fmt.Sprintf("v1/%s/reference-guide.md", system))
}

// LoadSkills composes core + system instruction files into a single skills
// document suitable for the prompt.
func (l *InstructionLoader) LoadSkills(system string) (string, error) {
	core, err := l.LoadCoreSkills()
	if err != nil {
		return "", fmt.Errorf("load core skills: %w", err)
	}

	var parts []string
	parts = append(parts, strings.TrimSpace(core))

	systemSkills, err := l.LoadSystemSkills(system)
	if err != nil {
		return "", fmt.Errorf("load system skills for %s: %w", system, err)
	}
	if text := strings.TrimSpace(systemSkills); text != "" {
		parts = append(parts, text)
	}

	memoryGuide, err := l.LoadCoreMemoryGuide()
	if err != nil {
		return "", fmt.Errorf("load core memory guide: %w", err)
	}
	if text := strings.TrimSpace(memoryGuide); text != "" {
		parts = append(parts, text)
	}

	refGuide, err := l.LoadSystemReferenceGuide(system)
	if err != nil {
		return "", fmt.Errorf("load system reference guide for %s: %w", system, err)
	}
	if text := strings.TrimSpace(refGuide); text != "" {
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n"), nil
}

func (l *InstructionLoader) readFile(path string) (string, error) {
	data, err := fs.ReadFile(l.fs, filepath.ToSlash(path))
	if err != nil {
		return "", fmt.Errorf("read instruction %s: %w", path, err)
	}
	return string(data), nil
}

func (l *InstructionLoader) readOptionalFile(path string) (string, error) {
	data, err := fs.ReadFile(l.fs, filepath.ToSlash(path))
	if err != nil {
		if isNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read instruction %s: %w", path, err)
	}
	return string(data), nil
}

func isNotExist(err error) bool {
	return os.IsNotExist(err) || strings.Contains(err.Error(), "file does not exist")
}
