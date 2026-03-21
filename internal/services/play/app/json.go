package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxJSONBodyBytes = 64 * 1024

var (
	// errJSONBodyTooLarge reports that the request body exceeded the permitted
	// size before JSON decoding began.
	errJSONBodyTooLarge = errors.New("request body exceeds maximum allowed size")
	// errJSONMultipleValues reports that the request body contained more than
	// one JSON value where exactly one was expected.
	errJSONMultipleValues = errors.New("request body must contain exactly one JSON value")
)

var protoJSONOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: true,
}

func decodeStrictJSON(r *http.Request, target any) error {
	if target == nil {
		return nil
	}
	if r == nil || r.Body == nil {
		return errJSONBodyTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	if len(body) > maxJSONBodyBytes {
		return errJSONBodyTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errJSONMultipleValues
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func marshalProtoJSON(message proto.Message) (json.RawMessage, error) {
	if message == nil {
		return json.RawMessage(`{}`), nil
	}
	data, err := protoJSONOptions.Marshal(message)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func writeProtoJSON(w http.ResponseWriter, status int, message proto.Message) error {
	data, err := marshalProtoJSON(message)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}
