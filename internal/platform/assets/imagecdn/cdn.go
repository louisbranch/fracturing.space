package imagecdn

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

var (
	ErrBaseURLRequired = errors.New("image cdn base url is required")
	ErrAssetIDRequired = errors.New("image asset id is required")
)

// ImageCDN resolves provider-specific asset URLs from provider-agnostic inputs.
type ImageCDN interface {
	URL(request Request) (string, error)
}

// Request describes one asset URL resolution request.
type Request struct {
	AssetID   string
	Extension string
	Crop      *Crop
	Delivery  *Delivery
}

// Crop defines one crop region in source-pixel coordinates.
type Crop struct {
	X        int
	Y        int
	WidthPX  int
	HeightPX int
}

// Delivery defines view-targeted delivery constraints.
type Delivery struct {
	WidthPX int
}

type flatCDN struct {
	baseURL string
}

type cloudinaryCDN struct {
	baseURL string
}

// New returns a platform ImageCDN resolver for the configured base URL.
//
// If the URL points to Cloudinary image/upload, Cloudinary transforms are used.
// Otherwise, a flat resolver joins base URL and filename.
func New(baseURL string) ImageCDN {
	normalizedBaseURL := strings.TrimSpace(baseURL)
	if isCloudinaryUploadBaseURL(normalizedBaseURL) {
		return cloudinaryCDN{baseURL: normalizedBaseURL}
	}
	return flatCDN{baseURL: normalizedBaseURL}
}

func (cdn flatCDN) URL(request Request) (string, error) {
	return resolveAssetURL(cdn.baseURL, request.AssetID, normalizeExtension(request.Extension), nil)
}

func (cdn cloudinaryCDN) URL(request Request) (string, error) {
	transforms := []string{}
	if crop := request.Crop; crop != nil && crop.WidthPX > 0 && crop.HeightPX > 0 {
		transforms = append(transforms, formatCloudinaryCrop(*crop))
	}
	if delivery := request.Delivery; delivery != nil && delivery.WidthPX > 0 {
		transforms = append(transforms, formatCloudinaryDelivery(*delivery))
	}
	return resolveAssetURL(cdn.baseURL, request.AssetID, normalizeExtension(request.Extension), transforms)
}

func resolveAssetURL(baseURL, assetID, extension string, transforms []string) (string, error) {
	normalizedBaseURL := strings.TrimSpace(baseURL)
	if normalizedBaseURL == "" {
		return "", ErrBaseURLRequired
	}
	normalizedAssetID := strings.TrimSpace(assetID)
	if normalizedAssetID == "" {
		return "", ErrAssetIDRequired
	}

	parsed, err := url.Parse(normalizedBaseURL)
	if err != nil {
		return "", err
	}

	pathSegments := append([]string(nil), transforms...)
	pathSegments = append(pathSegments, url.PathEscape(normalizedAssetID)+extension)
	parsed.Path = path.Join(parsed.Path, path.Join(pathSegments...))
	return parsed.String(), nil
}

func normalizeExtension(raw string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ".png"
	}
	if !strings.HasPrefix(normalized, ".") {
		return "." + normalized
	}
	return normalized
}

func formatCloudinaryCrop(crop Crop) string {
	return fmt.Sprintf(
		"c_crop,w_%d,h_%d,x_%d,y_%d",
		crop.WidthPX,
		crop.HeightPX,
		crop.X,
		crop.Y,
	)
}

func formatCloudinaryDelivery(delivery Delivery) string {
	return "f_auto,q_auto,dpr_auto,c_limit,w_" + strconv.Itoa(delivery.WidthPX)
}

func isCloudinaryUploadBaseURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Host, "res.cloudinary.com") {
		return false
	}
	pathLower := strings.ToLower(strings.TrimSpace(parsed.Path))
	return strings.Contains(pathLower, "/image/upload")
}
