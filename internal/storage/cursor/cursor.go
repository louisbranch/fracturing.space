// Package cursor provides opaque pagination token encoding/decoding.
package cursor

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Direction indicates the pagination direction.
type Direction string

const (
	// DirectionForward paginates forward (seq > cursor).
	DirectionForward Direction = "fwd"
	// DirectionBackward paginates backward (seq < cursor).
	DirectionBackward Direction = "bwd"
)

// Cursor represents the internal state of a pagination cursor.
type Cursor struct {
	// Seq is the sequence number to paginate from.
	Seq uint64 `json:"seq"`
	// Dir is the pagination direction (fwd = seq > cursor, bwd = seq < cursor).
	Dir Direction `json:"dir"`
	// Reverse indicates whether to temporarily reverse sort order.
	// This is needed when going to a "previous" page to fetch from the near edge.
	Reverse bool `json:"rev,omitempty"`
	// FilterHash ensures tokens are invalidated if the filter changes.
	FilterHash string `json:"filter_hash,omitempty"`
	// OrderHash ensures tokens are invalidated if the order_by changes.
	OrderHash string `json:"order_hash,omitempty"`
}

// Encode encodes a cursor to an opaque base64 string.
func Encode(c Cursor) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// Decode decodes an opaque base64 string to a cursor.
// Returns an error if the token is invalid or malformed.
func Decode(token string) (Cursor, error) {
	if token == "" {
		return Cursor{}, fmt.Errorf("empty token")
	}

	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode base64: %w", err)
	}

	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
	}

	// Validate direction
	if c.Dir != DirectionForward && c.Dir != DirectionBackward {
		return Cursor{}, fmt.Errorf("invalid cursor direction: %q", c.Dir)
	}

	return c, nil
}

// HashFilter computes a short hash of the filter string for cursor validation.
// Returns empty string for empty filter.
func HashFilter(filter string) string {
	if filter == "" {
		return ""
	}
	h := sha256.Sum256([]byte(filter))
	return hex.EncodeToString(h[:8]) // 64-bit hash is sufficient for validation
}

// ValidateFilterHash checks if the cursor's filter hash matches the current filter.
// Returns an error if the filter has changed since the cursor was created.
func ValidateFilterHash(c Cursor, currentFilter string) error {
	currentHash := HashFilter(currentFilter)
	if c.FilterHash != currentHash {
		return fmt.Errorf("filter changed since cursor was created")
	}
	return nil
}

// ValidateOrderHash checks if the cursor's order hash matches the current order_by.
// Returns an error if the order_by has changed since the cursor was created.
func ValidateOrderHash(c Cursor, currentOrderBy string) error {
	currentHash := HashFilter(currentOrderBy) // Reuse same hashing logic
	if c.OrderHash != currentHash {
		return fmt.Errorf("order_by changed since cursor was created")
	}
	return nil
}

// NewForwardCursor creates a cursor for forward pagination (seq > cursor) from the given sequence.
func NewForwardCursor(seq uint64, filter, orderBy string) Cursor {
	return Cursor{
		Seq:        seq,
		Dir:        DirectionForward,
		Reverse:    false,
		FilterHash: HashFilter(filter),
		OrderHash:  HashFilter(orderBy),
	}
}

// NewBackwardCursor creates a cursor for backward pagination (seq < cursor) from the given sequence.
func NewBackwardCursor(seq uint64, filter, orderBy string) Cursor {
	return Cursor{
		Seq:        seq,
		Dir:        DirectionBackward,
		Reverse:    false,
		FilterHash: HashFilter(filter),
		OrderHash:  HashFilter(orderBy),
	}
}

// NewNextPageCursor creates a cursor for the next page.
// For ASC order: seq > lastSeq (forward)
// For DESC order: seq < lastSeq (backward)
func NewNextPageCursor(lastSeq uint64, descending bool, filter, orderBy string) Cursor {
	dir := DirectionForward
	if descending {
		dir = DirectionBackward
	}
	return Cursor{
		Seq:        lastSeq,
		Dir:        dir,
		Reverse:    false,
		FilterHash: HashFilter(filter),
		OrderHash:  HashFilter(orderBy),
	}
}

// NewPrevPageCursor creates a cursor for the previous page.
// For ASC order: seq < firstSeq (backward), with temp reverse to get nearest items
// For DESC order: seq > firstSeq (forward), with temp reverse to get nearest items
func NewPrevPageCursor(firstSeq uint64, descending bool, filter, orderBy string) Cursor {
	dir := DirectionBackward
	if descending {
		dir = DirectionForward
	}
	return Cursor{
		Seq:        firstSeq,
		Dir:        dir,
		Reverse:    true,
		FilterHash: HashFilter(filter),
		OrderHash:  HashFilter(orderBy),
	}
}
