package app

import (
	"encoding/json"
	"errors"
	"html/template"
	"path/filepath"
	"strings"

	playui "github.com/louisbranch/fracturing.space/internal/services/play/ui"
)

type shellAssets struct {
	devServerURL string
	entryJS      string
	entryCSS     string
}

type shellRenderInput struct {
	CampaignID    string
	BootstrapPath string
	RealtimePath  string
	BackURL       string
}

type shellTemplateData struct {
	UIDevServerURL string
	UseDevServer   bool
	EntryJS        string
	EntryCSS       string
	ShellJSON      template.JS
}

type viteManifestEntry struct {
	File string   `json:"file"`
	CSS  []string `json:"css"`
}

var shellTemplate = template.Must(template.New("play-shell").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>Play</title>
    {{- if .UseDevServer }}
    <script type="module" src="{{ .UIDevServerURL }}/@vite/client"></script>
    <script type="module" src="{{ .UIDevServerURL }}/src/main.tsx"></script>
    {{- else }}
    {{- if .EntryCSS }}
    <link rel="stylesheet" href="{{ .EntryCSS }}"/>
    {{- end }}
    <script type="module" src="{{ .EntryJS }}"></script>
    {{- end }}
  </head>
  <body>
    <div id="root"></div>
    <script id="play-shell-config" type="application/json">{{ .ShellJSON }}</script>
  </body>
</html>
`))

func loadShellAssets(devServerURL string) (shellAssets, error) {
	devServerURL = strings.TrimRight(strings.TrimSpace(devServerURL), "/")
	if devServerURL != "" {
		return shellAssets{devServerURL: devServerURL}, nil
	}
	manifestBytes, err := playui.DistFS.ReadFile("dist/manifest.json")
	if err != nil {
		return shellAssets{}, errors.New("play ui manifest is missing; set FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL or build internal/services/play/ui")
	}
	manifest := map[string]viteManifestEntry{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return shellAssets{}, err
	}
	entry, ok := manifest["index.html"]
	if !ok || strings.TrimSpace(entry.File) == "" {
		return shellAssets{}, errors.New("play ui manifest is missing the index.html entry")
	}
	assets := shellAssets{
		entryJS: "/assets/play/" + filepath.ToSlash(strings.TrimSpace(entry.File)),
	}
	if len(entry.CSS) > 0 && strings.TrimSpace(entry.CSS[0]) != "" {
		assets.entryCSS = "/assets/play/" + filepath.ToSlash(strings.TrimSpace(entry.CSS[0]))
	}
	return assets, nil
}

func (a shellAssets) renderHTML(input shellRenderInput) ([]byte, error) {
	payload := map[string]string{
		"campaign_id":    strings.TrimSpace(input.CampaignID),
		"bootstrap_path": strings.TrimSpace(input.BootstrapPath),
		"realtime_path":  strings.TrimSpace(input.RealtimePath),
		"back_url":       strings.TrimSpace(input.BackURL),
	}
	shellJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	if err := shellTemplate.Execute(&builder, shellTemplateData{
		UIDevServerURL: a.devServerURL,
		UseDevServer:   a.devServerURL != "",
		EntryJS:        a.entryJS,
		EntryCSS:       a.entryCSS,
		ShellJSON:      template.JS(shellJSON),
	}); err != nil {
		return nil, err
	}
	return []byte(builder.String()), nil
}
