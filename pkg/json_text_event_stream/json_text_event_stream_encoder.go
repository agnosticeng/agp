package json_text_event_stream

import (
	"encoding/json"
	"fmt"
	"io"
)

type JSONTextEventStreamEncoder struct {
	w io.Writer
}

func NewJSONTextEventStreamEncoder(w io.Writer) *JSONTextEventStreamEncoder {
	return &JSONTextEventStreamEncoder{w: w}
}

func (enc *JSONTextEventStreamEncoder) Encode(event string, v any) error {
	var prelude string

	if len(event) == 0 {
		prelude = "data: "
	} else {
		prelude = fmt.Sprintf("event: %s\ndata: ", event)
	}

	if _, err := enc.w.Write([]byte(prelude)); err != nil {
		return err
	}

	if err := json.NewEncoder(enc.w).Encode(v); err != nil {
		return err
	}

	if _, err := enc.w.Write([]byte("\n\n")); err != nil {
		return err
	}

	return nil
}
