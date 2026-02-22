// Package profile validates and normalizes public profile inputs.
package profile

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	maxDisplayNameLength = 64
	maxAvatarURLLength   = 512
	maxBioLength         = 280
)

// Normalized stores validated profile field values.
type Normalized struct {
	DisplayName string
	AvatarURL   string
	Bio         string
}

// Normalize validates and trims user-supplied public profile values.
func Normalize(displayName string, avatarURL string, bio string) (Normalized, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return Normalized{}, fmt.Errorf("display name is required")
	}
	if utf8.RuneCountInString(displayName) > maxDisplayNameLength {
		return Normalized{}, fmt.Errorf("display name must be at most %d characters", maxDisplayNameLength)
	}

	avatarURL = strings.TrimSpace(avatarURL)
	if len(avatarURL) > maxAvatarURLLength {
		return Normalized{}, fmt.Errorf("avatar url must be at most %d characters", maxAvatarURLLength)
	}
	if avatarURL != "" {
		parsed, err := url.Parse(avatarURL)
		if err != nil || !parsed.IsAbs() {
			return Normalized{}, fmt.Errorf("avatar url must be an absolute URL")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return Normalized{}, fmt.Errorf("avatar url scheme must be http or https")
		}
		avatarURL = parsed.String()
	}

	bio = strings.TrimSpace(bio)
	if utf8.RuneCountInString(bio) > maxBioLength {
		return Normalized{}, fmt.Errorf("bio must be at most %d characters", maxBioLength)
	}

	return Normalized{
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Bio:         bio,
	}, nil
}
