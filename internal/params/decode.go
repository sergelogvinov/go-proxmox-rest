package params

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Decode is the reverse of Encode: it unmarshals a Proxmox API JSON payload
// into out (a non-nil pointer), recursing into nested structs and slices of
// structs using the `json` struct tags already carried by every response
// DTO.
//
// It behaves like encoding/json for every field except one: a field of type
// []string is, like Encode's comma-join, allowed to arrive on the wire as a
// single JSON string (e.g. "content":"images,iso,vztmpl") rather than a JSON
// array. Decode splits that string on "," into the slice. A field already
// sent as a genuine JSON array decodes exactly as encoding/json would.
func Decode(data []byte, out any) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("params: decode target must be a non-nil pointer")
	}

	return decodeValue(data, rv.Elem())
}

// decodeValue decodes raw into the addressable value rv, recursing into
// structs and slices of structs so nested DTOs get the same []string
// handling as their parent.
func decodeValue(raw json.RawMessage, rv reflect.Value) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	if rv.Kind() == reflect.Pointer {
		rv.Set(reflect.New(rv.Type().Elem()))
		return decodeValue(raw, rv.Elem())
	}

	switch {
	case rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.String && raw[0] == '"':
		return decodeCommaList(raw, rv)

	case rv.Kind() == reflect.Struct:
		return decodeStruct(raw, rv)

	case rv.Kind() == reflect.Slice && elemIsStruct(rv.Type().Elem()):
		return decodeStructSlice(raw, rv)

	default:
		return json.Unmarshal(raw, rv.Addr().Interface())
	}
}

// decodeStruct decodes a JSON object into rv field-by-field, matching each
// field's `json` tag name.
func decodeStruct(raw json.RawMessage, rv reflect.Value) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		name, ok := jsonName(f.Tag.Get("json"), f.Name)
		if !ok {
			continue
		}

		item, present := m[name]
		if !present {
			continue
		}

		if err := decodeValue(item, rv.Field(i)); err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
	}

	return nil
}

// decodeStructSlice decodes a JSON array of objects into rv, applying
// decodeValue (and therefore struct/[]string handling) to each element.
func decodeStructSlice(raw json.RawMessage, rv reflect.Value) error {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}

	elemType := rv.Type().Elem()
	out := reflect.MakeSlice(rv.Type(), len(items), len(items))

	for i, item := range items {
		ev := out.Index(i)
		if elemType.Kind() == reflect.Pointer {
			ev.Set(reflect.New(elemType.Elem()))
			ev = ev.Elem()
		}
		if err := decodeValue(item, ev); err != nil {
			return err
		}
	}

	rv.Set(out)

	return nil
}

// decodeCommaList splits a JSON string on "," into rv, mirroring Encode's
// comma-join for []string fields.
func decodeCommaList(raw json.RawMessage, rv reflect.Value) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := reflect.MakeSlice(rv.Type(), len(parts), len(parts))
	for i, p := range parts {
		out.Index(i).SetString(p)
	}
	rv.Set(out)

	return nil
}

// elemIsStruct reports whether t (or the type it points to) is a struct.
func elemIsStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// jsonName extracts the wire field name from a `json:"..."` tag, reporting
// whether the field should be decoded at all (mirrors encoding/json: a "-"
// tag skips the field; a missing/empty name falls back to the Go field
// name).
func jsonName(tag, fieldName string) (string, bool) {
	if tag == "-" {
		return "", false
	}

	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = fieldName
	}

	return name, true
}
