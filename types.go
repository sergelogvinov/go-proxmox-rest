package proxmox

import (
	"encoding/json"
)

// envelope is the Proxmox response wrapper: { "data": ..., "errors": ... }.
type envelope struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

// decodeInto unwraps the { "data": ... } envelope and decodes the payload
// into out. A null data yields the zero value of out. Unknown fields are
// ignored so the API can grow without breaking this client.
func decodeInto[T any](b []byte, out T) error {
	var env envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return err
	}

	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}

	if err := json.Unmarshal(env.Data, out); err != nil {
		return err
	}

	return nil
}
