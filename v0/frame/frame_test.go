package frame_test

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v0/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v0/frame"
	"github.com/dejitarudemon/ignicula-wire/v0/internal/specs"
	"github.com/dejitarudemon/ignicula-wire/v0/internal/testutil"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

func encode(t *testing.T, f frame.Frame) []byte {
	t.Helper()

	buf := buffer.Slice{}
	buf.Preallocate(f.Size())

	if e := f.Encode(&buf); e != nil {
		t.Fatalf("Encode() = %v", e)
	}

	return buf.Bytes()
}

func TestFrame_Specs(t *testing.T) {
	for _, tt := range specs.Frames {
		t.Run(tt.Name, func(t *testing.T) {
			if got := encode(t, tt.Frame); !bytes.Equal(got, tt.Encoded) {
				t.Errorf("Encode() = % X, want % X", got, tt.Encoded)
			}

			if got := tt.Frame.Size(); got != len(tt.Encoded) {
				t.Errorf("Size() = %v, want %v", got, len(tt.Encoded))
			}

			if got := tt.Frame.Version(); got != fields.CurrentVersion {
				t.Errorf("Version() = %v, want %v", got, fields.CurrentVersion)
			}

			if e := tt.Frame.IsValid(); e != nil {
				t.Errorf("IsValid() = %v", e)
			}
		})
	}
}

func TestFrame_Encode(t *testing.T) {
	tests := []struct {
		name string
		f    frame.Frame
	}{
		{"empty", frame.Frame{Body: bodies.Hello{}}},
		{"one version", frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1}}}},
		{"three versions", frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1, 2, 3}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encode(t, tt.f)

			if !bytes.Equal(got[:2], frame.MagicBytes) {
				t.Errorf("magic = % X, want % X", got[:2], frame.MagicBytes)
			}

			if got[2] != byte(fields.CurrentVersion) {
				t.Errorf("version = %v, want %v", got[2], fields.CurrentVersion)
			}

			body := testutil.Encode(t, tt.f.Body)
			if got[3] != byte(len(body)) {
				t.Errorf("version len = %v, want %v", got[3], len(body))
			}

			if !bytes.Equal(got[4:4+len(body)], body) {
				t.Errorf("body = % X, want % X", got[4:4+len(body)], body)
			}

			checksum := got[len(got)-4:]
			if want := fields.NewChecksum(got[:len(got)-4]); binary.BigEndian.Uint32(checksum) != uint32(want) {
				t.Errorf("checksum = % X, want %08X", checksum, uint32(want))
			}

			if got := tt.f.Size(); got != len(encode(t, tt.f)) {
				t.Errorf("Size() = %v, want encoded length", got)
			}
		})
	}
}

func TestFrame_EncodeErrs(t *testing.T) {
	buf := buffer.Slice{}

	if e := (frame.Frame{}).Encode(&buf); e == nil {
		t.Fatal("Encode() = nil, want error")
	}
}

func TestFrame_NilBody(t *testing.T) {
	f := frame.Frame{}

	if got, want := f.Size(), 8; got != want {
		t.Errorf("Size() = %v, want %v", got, want)
	}

	if e := f.IsValid(); e == nil {
		t.Error("IsValid() = nil, want error")
	}
}

func TestFrame_IsValid(t *testing.T) {
	tooMany := slices.Repeat([]fields.Version{1}, bodies.MaxVersionsPerOneHello+1)

	tests := []struct {
		name string
		f    frame.Frame
	}{
		{"nil body", frame.Frame{}},
		{"version 0", frame.Frame{Body: bodies.Hello{Versions: []fields.Version{0}}}},
		{"too many versions", frame.Frame{Body: bodies.Hello{Versions: tooMany}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if e := tt.f.IsValid(); e == nil {
				t.Error("IsValid() = nil, want error")
			}
		})
	}
}
