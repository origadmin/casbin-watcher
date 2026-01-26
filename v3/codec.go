package watcher

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// MarshalUnmarshaler is the interface for serializing and deserializing UpdateMessage.
type MarshalUnmarshaler interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

// gobMarshalUnmarshaler provides a default gob implementation for MarshalUnmarshaler.
type gobMarshalUnmarshaler struct{}

// Marshal uses gob to marshal the value.
func (g *gobMarshalUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal uses gob to unmarshal the data.
func (g *gobMarshalUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}

// jsonMarshalUnmarshaler provides a JSON implementation for MarshalUnmarshaler.
type jsonMarshalUnmarshaler struct{}

// Marshal uses JSON to marshal the value.
func (j *jsonMarshalUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal uses JSON to unmarshal the data.
func (j *jsonMarshalUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// DefaultCodec returns the default gob codec.
func DefaultCodec() MarshalUnmarshaler {
	return &gobMarshalUnmarshaler{}
}

// JSONCodec returns a JSON codec.
func JSONCodec() MarshalUnmarshaler {
	return &jsonMarshalUnmarshaler{}
}
