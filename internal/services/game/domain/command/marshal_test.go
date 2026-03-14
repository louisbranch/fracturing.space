package command

import (
	"testing"
)

func TestMustMarshalJSON_Success(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	data := MustMarshalJSON(payload{Name: "test"})
	if string(data) != `{"name":"test"}` {
		t.Fatalf("MustMarshalJSON() = %s, want %s", data, `{"name":"test"}`)
	}
}

func TestMustMarshalJSON_PanicsOnUnsafeType(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustMarshalJSON(func) should panic")
		}
	}()
	// Functions cannot be marshalled to JSON.
	MustMarshalJSON(func() {})
}
