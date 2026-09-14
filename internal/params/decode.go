package params

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Decode is the reverse of Encode: it unmarshals a Proxmox API JSON payload
// into out (a non-nil pointer), recursing into nested structs and slices of
// structs using the same `url` struct tags Encode uses, so a single tag per
// field names it on the wire for both directions.
//
// It behaves like encoding/json for every field except two:
//
//   - A field of type []string is, like Encode's comma-join, allowed to
//     arrive on the wire as a single JSON string (e.g.
//     "content":"images,iso,vztmpl") rather than a JSON array. Decode splits
//     that string on "," into the slice. A field already sent as a genuine
//     JSON array decodes exactly as encoding/json would.
//   - A field of type bool (or *bool) is allowed to arrive on the wire as a
//     JSON number (1/0) or a JSON string ("1"/"0"/"true"/"false"), in
//     addition to a genuine JSON bool, matching Proxmox's inconsistent
//     encoding of boolean fields across endpoints.
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

	case rv.Kind() == reflect.Bool:
		return decodeBool(raw, rv)

	case rv.Kind() == reflect.Struct:
		return decodeStruct(raw, rv)

	case rv.Kind() == reflect.Slice && elemIsStruct(rv.Type().Elem()):
		return decodeStructSlice(raw, rv)

	default:
		return json.Unmarshal(raw, rv.Addr().Interface())
	}
}

// decodeStruct decodes a JSON object into rv field-by-field, matching each
// field's `url` tag name.
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

		name, ok := urlName(f.Tag.Get("url"), f.Name)
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

// decodeBool decodes raw into the bool rv, accepting a genuine JSON bool, a
// JSON number (nonzero is true), or a JSON string ("1"/"0"/"true"/"false"),
// since Proxmox encodes boolean fields inconsistently across endpoints.
func decodeBool(raw json.RawMessage, rv reflect.Value) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}

	switch t := v.(type) {
	case bool:
		rv.SetBool(t)

	case float64:
		rv.SetBool(t != 0)

	case string:
		switch strings.ToLower(t) {
		case "1", "true":
			rv.SetBool(true)
		case "0", "false", "":
			rv.SetBool(false)
		default:
			return fmt.Errorf("params: cannot decode %q as bool", t)
		}

	default:
		return fmt.Errorf("params: cannot decode %T as bool", v)
	}

	return nil
}

// elemIsStruct reports whether t (or the type it points to) is a struct.
func elemIsStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// urlName extracts the wire field name from a `url:"..."` tag, reporting
// whether the field should be decoded at all: a missing or "-" tag skips
// the field, since (unlike encoding/json) an untagged field carries no wire
// name to decode by.
func urlName(tag, fieldName string) (string, bool) {
	if tag == "" || tag == "-" {
		return "", false
	}

	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = fieldName
	}

	return name, true
}
