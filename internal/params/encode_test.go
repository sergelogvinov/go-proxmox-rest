package params

import (
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	shared := true
	disabled := false
	maxFiles := 0
	comment := ""
	empty := ""

	cases := []struct {
		name string
		opts any
		want map[string]string
	}{
		{
			name: "all field kinds",
			opts: struct {
				ID       string   `url:"storage"`
				Type     string   `url:"type"`
				Content  []string `url:"content"`
				Shared   *bool    `url:"shared"`
				Disabled *bool    `url:"disable"`
				MaxFiles *int     `url:"maxfiles"`
				Comment  *string  `url:"comment"`
				NoTag    string
				Skipped  string `url:"-"`
			}{
				ID:       "local-zfs",
				Type:     "zfs",
				Content:  []string{"images", "iso"},
				Shared:   &shared,
				Disabled: &disabled,
				MaxFiles: &maxFiles,
				Comment:  &comment,
				NoTag:    "ignored",
				Skipped:  "ignored",
			},
			want: map[string]string{
				"storage":  "local-zfs",
				"type":     "zfs",
				"content":  "images,iso",
				"shared":   "1",
				"disable":  "0",
				"maxfiles": "0",
				"comment":  "",
			},
		},
		{
			name: "zero values omitted",
			opts: struct {
				Name  string   `url:"name"`
				Count int      `url:"count"`
				On    bool     `url:"on"`
				List  []string `url:"list"`
			}{},
			want: map[string]string{},
		},
		{
			name: "nil pointers omitted",
			opts: struct {
				Comment *string `url:"comment"`
			}{},
			want: map[string]string{},
		},
		{
			name: "pointer to struct",
			opts: &struct {
				Name string `url:"name"`
			}{Name: "pve"},
			want: map[string]string{"name": "pve"},
		},
		{
			name: "empty string value",
			opts: struct {
				Name string `url:"name"`
			}{Name: empty},
			want: map[string]string{},
		},
		{
			name: "readonly field is never sent, even non-zero",
			opts: struct {
				Name   string `url:"name"`
				Digest string `url:"digest,readonly"`
			}{Name: "pve", Digest: "abc123"},
			want: map[string]string{"name": "pve"},
		},
		{
			name: "writeonly modifier has no effect on Encode",
			opts: struct {
				Rename string `url:"rename,writeonly"`
			}{Rename: "old-name"},
			want: map[string]string{"rename": "old-name"},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := Encode(test.opts)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() = %v, want %v", got, test.want)
			}
		})
	}
}
