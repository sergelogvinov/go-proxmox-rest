package params

import (
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	type member struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	type storage struct {
		ID      string   `json:"storage"`
		Content []string `json:"content"`
		Shared  int      `json:"shared"`
		Ignored string   `json:"-"`
	}

	t.Run("comma-joined string decodes to []string", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{"storage":"local","content":"images,iso,vztmpl","shared":1}`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		want := storage{ID: "local", Content: []string{"images", "iso", "vztmpl"}, Shared: 1, Ignored: ""}
		if !reflect.DeepEqual(s, want) {
			t.Errorf("Decode() = %+v, want %+v", s, want)
		}
	})

	t.Run("empty comma string yields empty slice", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{"storage":"local","content":""}`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if s.Content != nil {
			t.Errorf("Content = %v, want nil", s.Content)
		}
	})

	t.Run("genuine JSON array still decodes", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{"storage":"local","content":["images","iso"]}`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		want := []string{"images", "iso"}
		if !reflect.DeepEqual(s.Content, want) {
			t.Errorf("Content = %v, want %v", s.Content, want)
		}
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{"storage":"local","future_field":"x"}`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if s.ID != "local" {
			t.Errorf("ID = %q, want %q", s.ID, "local")
		}
	})

	t.Run("json:- field is skipped", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{"Ignored":"x"}`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if s.Ignored != "" {
			t.Errorf("Ignored = %q, want empty", s.Ignored)
		}
	})

	t.Run("null data leaves zero value", func(t *testing.T) {
		s := storage{ID: "keep", Content: nil, Shared: 0, Ignored: ""}
		if err := Decode([]byte(`null`), &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if s.ID != "keep" {
			t.Errorf("ID = %q, want %q (unchanged)", s.ID, "keep")
		}
	})

	t.Run("nested struct slice with comma field", func(t *testing.T) {
		type pool struct {
			Members []member `json:"members"`
		}

		var p pool
		data := []byte(`{"members":[{"name":"vm/100","tags":"a,b"},{"name":"vm/101","tags":""}]}`)
		if err := Decode(data, &p); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		want := pool{Members: []member{
			{Name: "vm/100", Tags: []string{"a", "b"}},
			{Name: "vm/101", Tags: nil},
		}}
		if !reflect.DeepEqual(p, want) {
			t.Errorf("Decode() = %+v, want %+v", p, want)
		}
	})

	t.Run("top-level slice of structs", func(t *testing.T) {
		var storages []storage
		data := []byte(`[{"storage":"a","content":"iso"},{"storage":"b","content":"images,iso"}]`)
		if err := Decode(data, &storages); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		want := []storage{
			{ID: "a", Content: []string{"iso"}, Shared: 0, Ignored: ""},
			{ID: "b", Content: []string{"images", "iso"}, Shared: 0, Ignored: ""},
		}
		if !reflect.DeepEqual(storages, want) {
			t.Errorf("Decode() = %+v, want %+v", storages, want)
		}
	})

	t.Run("non-pointer target errors", func(t *testing.T) {
		var s storage
		if err := Decode([]byte(`{}`), s); err == nil {
			t.Fatal("Decode() error = nil, want error")
		}
	})

	t.Run("empty data is a no-op", func(t *testing.T) {
		var s storage
		if err := Decode(nil, &s); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
	})
}
