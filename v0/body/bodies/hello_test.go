package bodies

import (
	"slices"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v0/internal/testutil"
)

func versions(n int, first fields.Version) []fields.Version {
	out := make([]fields.Version, n)
	for i := range out {
		out[i] = first + fields.Version(i)
	}
	return out
}

func encodeVersions(vs []fields.Version) []byte {
	n := min(len(vs), MaxVersionsPerOneHello)
	out := make([]byte, n)
	for i, v := range vs[:n] {
		out[i] = byte(v)
	}
	return out
}

func TestHello(t *testing.T) {
	tests := []struct {
		name    string
		b       Hello
		want    []byte
		wantErr bool
	}{
		{"empty", Hello{}, nil, false},
		{"nil versions", Hello{Versions: nil}, nil, false},
		{"one version", Hello{Versions: []fields.Version{1}}, []byte{0x01}, false},
		{"client spec", Hello{Versions: []fields.Version{1, 2, 3}}, []byte{0x01, 0x02, 0x03}, false},
		{"server spec", Hello{Versions: []fields.Version{1, 4, 7, 11}}, []byte{0x01, 0x04, 0x07, 0x0B}, false},
		{"duplicates", Hello{Versions: []fields.Version{1, 1, 2}}, []byte{0x01, 0x01, 0x02}, false},
		{"version 0", Hello{Versions: []fields.Version{0, 1}}, []byte{0x00, 0x01}, true},
		{"only version 0", Hello{Versions: []fields.Version{0}}, []byte{0x00}, true},
		{"max versions", Hello{Versions: versions(MaxVersionsPerOneHello, 1)}, encodeVersions(versions(MaxVersionsPerOneHello, 1)), false},
		{"too many versions", Hello{Versions: versions(MaxVersionsPerOneHello+1, 1)}, encodeVersions(versions(MaxVersionsPerOneHello+1, 1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertEncoded(t, tt.b, tt.want)
			testutil.AssertErr(t, tt.b.IsValid(), tt.wantErr)
		})
	}
}

func TestNewHello(t *testing.T) {
	tests := []struct {
		name string
		got  Hello
		want Hello
	}{
		{"nil", NewHello(nil), Hello{Versions: []fields.Version{}}},
		{"empty", NewHello([]fields.Version{}), Hello{Versions: []fields.Version{}}},
		{"keeps order", NewHello([]fields.Version{3, 1, 2}), Hello{Versions: []fields.Version{3, 1, 2}}},
		{"drops zero", NewHello([]fields.Version{0, 1, 2}), Hello{Versions: []fields.Version{1, 2}}},
		{"drops duplicates", NewHello([]fields.Version{1, 1, 2, 1}), Hello{Versions: []fields.Version{1, 2}}},
		{"all unique working versions", NewHello(versions(MaxVersionsPerOneHello, 1)), Hello{Versions: versions(MaxVersionsPerOneHello, 1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got.Versions, tt.want.Versions) {
				t.Errorf("Versions = %v, want %v", tt.got.Versions, tt.want.Versions)
			}

			testutil.AssertSameEncoding(t, tt.got, tt.want)
			testutil.AssertErr(t, tt.got.IsValid(), false)
		})
	}
}

func TestHello_Common(t *testing.T) {
	tests := []struct {
		name string
		a, b Hello
		want []fields.Version
	}{
		{"empty", Hello{}, Hello{Versions: []fields.Version{1}}, []fields.Version{}},
		{"none", Hello{Versions: []fields.Version{1}}, Hello{Versions: []fields.Version{2}}, []fields.Version{}},
		{"spec", Hello{Versions: []fields.Version{1, 2, 3}}, Hello{Versions: []fields.Version{1, 4, 7, 11}}, []fields.Version{1}},
		{"preserves receiver order", Hello{Versions: []fields.Version{3, 1, 2}}, Hello{Versions: []fields.Version{2, 3}}, []fields.Version{3, 2}},
		{"ignores zero", Hello{Versions: []fields.Version{0, 1}}, Hello{Versions: []fields.Version{0, 1, 2}}, []fields.Version{1}},
		{"ignores duplicates", Hello{Versions: []fields.Version{1, 1, 2}}, Hello{Versions: []fields.Version{1, 2, 2}}, []fields.Version{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Common(tt.b); !slices.Equal(got, tt.want) {
				t.Errorf("Common() = %v, want %v", got, tt.want)
			}
		})
	}
}
