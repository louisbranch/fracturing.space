---
title: "Icon Rendering"
parent: "Project"
nav_order: 31
---

# Icon Rendering

Core icon identifiers (`commonv1.IconId`) are mapped to Lucide icon names in `internal/platform/icons/lucide.go`. UI surfaces should render icons through the shared sprite so the admin and web services stay visually consistent.

## Source of truth

- Mapping: `internal/platform/icons/lucide.go`
- Sprite markup: `internal/platform/icons/lucide_sprite.go`
- Shared templ helpers: `internal/services/shared/templates/icons.templ`

## Usage

Inject the sprite once per page (already wired into admin and web layouts) and render icons with the shared helpers:

```templ
@sharedtemplates.LucideIcon("book-open", "size-4")
@sharedtemplates.LucideIconID(iconID, "size-4")
```

When adding a new core icon, update the mapping and sprite together so tests and rendering stay aligned.
