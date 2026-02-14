package pagination

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
	// DirectionForward paginates forward (value > cursor).
	DirectionForward Direction = "fwd"
	// DirectionBackward paginates backward (value < cursor).
	DirectionBackward Direction = "bwd"
)

// CursorValueKind indicates the cursor value type.
type CursorValueKind string

const (
	CursorValueString CursorValueKind = "string"
	CursorValueInt    CursorValueKind = "int"
	CursorValueUint   CursorValueKind = "uint"
)

// CursorValue represents a typed cursor value.
type CursorValue struct {
	Name        string          `json:"name"`
	Kind        CursorValueKind `json:"kind"`
	StringValue string          `json:"string,omitempty"`
	IntValue    int64           `json:"int,omitempty"`
	UintValue   uint64          `json:"uint,omitempty"`
}

// Cursor represents the internal state of a pagination cursor.
type Cursor struct {
	Values     []CursorValue `json:"values,omitempty"`
	Dir        Direction     `json:"dir"`
	Reverse    bool          `json:"rev,omitempty"`
	FilterHash string        `json:"filter_hash,omitempty"`
	OrderHash  string        `json:"order_hash,omitempty"`
}

// Value returns a cursor value by name.
func (c Cursor) Value(name string) (CursorValue, bool) {
	for _, value := range c.Values {
		if value.Name == name {
			return value, true
		}
	}
	return CursorValue{}, false
}

// ValueString returns a string cursor value by name.
func ValueString(c Cursor, name string) (string, error) {
	value, ok := c.Value(name)
	if !ok {
		return "", fmt.Errorf("cursor missing %s", name)
	}
	if value.Kind != CursorValueString {
		return "", fmt.Errorf("cursor value %s is not a string", name)
	}
	return value.StringValue, nil
}

// ValueInt returns an int cursor value by name.
func ValueInt(c Cursor, name string) (int64, error) {
	value, ok := c.Value(name)
	if !ok {
		return 0, fmt.Errorf("cursor missing %s", name)
	}
	if value.Kind != CursorValueInt {
		return 0, fmt.Errorf("cursor value %s is not an int", name)
	}
	return value.IntValue, nil
}

// ValueUint returns a uint cursor value by name.
func ValueUint(c Cursor, name string) (uint64, error) {
	value, ok := c.Value(name)
	if !ok {
		return 0, fmt.Errorf("cursor missing %s", name)
	}
	if value.Kind != CursorValueUint {
		return 0, fmt.Errorf("cursor value %s is not a uint", name)
	}
	return value.UintValue, nil
}

// StringValue creates a string cursor value.
func StringValue(name, value string) CursorValue {
	return CursorValue{Name: name, Kind: CursorValueString, StringValue: value}
}

// IntValue creates an int cursor value.
func IntValue(name string, value int64) CursorValue {
	return CursorValue{Name: name, Kind: CursorValueInt, IntValue: value}
}

// UintValue creates a uint cursor value.
func UintValue(name string, value uint64) CursorValue {
	return CursorValue{Name: name, Kind: CursorValueUint, UintValue: value}
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
	return hex.EncodeToString(h[:8])
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
	currentHash := HashFilter(currentOrderBy)
	if c.OrderHash != currentHash {
		return fmt.Errorf("order_by changed since cursor was created")
	}
	return nil
}

// NewCursor creates a cursor with the provided metadata.
func NewCursor(values []CursorValue, dir Direction, reverse bool, filter, orderBy string) Cursor {
	return Cursor{
		Values:     values,
		Dir:        dir,
		Reverse:    reverse,
		FilterHash: HashFilter(filter),
		OrderHash:  HashFilter(orderBy),
	}
}

// NewNextPageCursor creates a cursor for the next page.
// For ASC order: value > lastKey (forward)
// For DESC order: value < lastKey (backward)
func NewNextPageCursor(values []CursorValue, descending bool, filter, orderBy string) Cursor {
	dir := DirectionForward
	if descending {
		dir = DirectionBackward
	}
	return NewCursor(values, dir, false, filter, orderBy)
}

// NewPrevPageCursor creates a cursor for the previous page.
// For ASC order: value < firstKey (backward), with temp reverse to get nearest items
// For DESC order: value > firstKey (forward), with temp reverse to get nearest items
func NewPrevPageCursor(values []CursorValue, descending bool, filter, orderBy string) Cursor {
	dir := DirectionBackward
	if descending {
		dir = DirectionForward
	}
	return NewCursor(values, dir, true, filter, orderBy)
}
