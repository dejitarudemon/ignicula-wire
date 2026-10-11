package fields

import (
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/internal/testutil"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		v    Version
		want []byte
	}{
		{"zero", Version(0), []byte{0x00}},
		{"current", Version(CurrentVersion), []byte{0x01}},
		{"small", Version(3), []byte{0x03}},
		{"max byte", Version(255), []byte{0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertEncoded(t, tt.v, tt.want)
		})
	}
}
