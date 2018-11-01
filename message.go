package seriald

import (
	"encoding/json"
	"io"
)

type ResponseMessage struct {
	Error  string        `json:",omitempty"`
	Value  interface{}   `json:",omitempty"`
}

func WriteMessage(out io.Writer, message *ResponseMessage) (n int, err error) {
	data, _ := json.MarshalIndent(message, "", "  ")
	return out.Write(data)
}
