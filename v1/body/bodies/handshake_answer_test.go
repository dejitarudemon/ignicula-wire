package bodies

import (
	"slices"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/internal/testutil"
)

func TestHandshakeAnswer(t *testing.T) {
	tests := []struct {
		name    string
		b       HandshakeAnswer
		want    []byte
		wantErr bool
	}{
		{"empty", HandshakeAnswer{}, []byte{0x01, 0x00, 0x00, 0x00}, false},
		{"empty compressions", HandshakeAnswer{Compressions: []fields.Compression{}}, []byte{0x01, 0x00, 0x00, 0x00}, false},
		{"none", HandshakeAnswer{Compressions: []fields.Compression{fields.None}}, []byte{0x01, 0x00, 0x01, 0x00, 0x00}, false},
		{"two compressions", HandshakeAnswer{Compressions: []fields.Compression{fields.None, fields.Zstd}}, []byte{0x01, 0x00, 0x02, 0x00, 0x01, 0x00}, false},
		{"repeated compressions", HandshakeAnswer{Compressions: []fields.Compression{fields.None, fields.Zstd, fields.None}}, []byte{0x01, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00}, false},
		{"custom compression", HandshakeAnswer{Compressions: []fields.Compression{fields.None, fields.Zstd, 0xFF}}, []byte{0x01, 0x00, 0x03, 0x00, 0x01, 0xFF, 0x00}, false},
		{
			"with versions",
			HandshakeAnswer{
				Compressions: []fields.Compression{fields.Zstd},
				Versions:     []fields.Version{1},
			},
			[]byte{0x01, 0x00, 0x01, 0x01, 0x01, 0x01},
			false,
		},
		{
			"versions only",
			HandshakeAnswer{Versions: []fields.Version{1, 2}},
			[]byte{0x01, 0x00, 0x00, 0x02, 0x01, 0x02},
			false,
		},
		{
			"max compressions",
			HandshakeAnswer{Compressions: generateManyCompressions(MaxCompressionsPerOneHandshake)},
			append(append([]byte{0x01, 0x00}, encodeCompressions(generateManyCompressions(MaxCompressionsPerOneHandshake))...), 0x00),
			false,
		},
		{
			"too many compressions",
			HandshakeAnswer{Compressions: generateManyCompressions(1000)},
			append(append([]byte{0x01, 0x00}, encodeCompressions(generateManyCompressions(1000))...), 0x00),
			true,
		},
		{
			"too many versions",
			HandshakeAnswer{Versions: generateManyVersions(1000)},
			append([]byte{0x01, 0x00, 0x00}, encodeVersions(generateManyVersions(1000))...),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertAnswer(t, tt.b, fields.Handshake, tt.want, tt.wantErr)
		})
	}
}

func TestNewHandshakeAnswer(t *testing.T) {
	tests := []struct {
		name string
		got  HandshakeAnswer
		want HandshakeAnswer
	}{
		{"nil", NewHandshakeAnswer(nil, nil), HandshakeAnswer{}},
		{
			"drops none",
			NewHandshakeAnswer([]fields.Compression{0x01, 0x00, 0x02}, nil),
			HandshakeAnswer{Compressions: []fields.Compression{0x01, 0x02}},
		},
		{
			"drops none and duplicates",
			NewHandshakeAnswer([]fields.Compression{0x01, 0x00, 0x02, 0x01, 0x03}, nil),
			HandshakeAnswer{Compressions: []fields.Compression{0x01, 0x02, 0x03}},
		},
		{
			"drops duplicates",
			NewHandshakeAnswer([]fields.Compression{0x01, 0x01}, nil),
			HandshakeAnswer{Compressions: []fields.Compression{0x01}},
		},
		{
			"drops compression duplicates beyond limit",
			NewHandshakeAnswer(generateManyCompressions(1000), nil),
			HandshakeAnswer{Compressions: generateManyCompressions(254)},
		},
		{
			"keeps versions and drops zero",
			NewHandshakeAnswer([]fields.Compression{fields.Zstd}, []fields.Version{0, 1, 1, 2}),
			HandshakeAnswer{
				Compressions: []fields.Compression{fields.Zstd},
				Versions:     []fields.Version{1, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got.Compressions, tt.want.Compressions) {
				t.Errorf("Compressions = %v, want %v", tt.got.Compressions, tt.want.Compressions)
			}
			if !slices.Equal(tt.got.Versions, tt.want.Versions) {
				t.Errorf("Versions = %v, want %v", tt.got.Versions, tt.want.Versions)
			}

			testutil.AssertSameEncoding(t, tt.got, tt.want)
			testutil.AssertErr(t, tt.got.IsValid(), false)
		})
	}
}

func generateManyVersions(n int) []fields.Version {
	out := make([]fields.Version, n)
	for i := range out {
		out[i] = fields.Version(i%255 + 1)
	}
	return out
}

func encodeVersions(versions []fields.Version) []byte {
	n := min(len(versions), MaxVersionsPerOneHandshakeAnswer)
	out := make([]byte, 1+n)
	out[0] = byte(n)
	for i := range n {
		out[1+i] = byte(versions[i])
	}
	return out
}
