package builder

import (
	"slices"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v0/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v0/frame"
	"github.com/dejitarudemon/ignicula-wire/v0/internal/testutil"
)

func TestFrameBuilder(t *testing.T) {
	fb := NewFrameBuilder(1 << 10)
	tooManyOnes := slices.Repeat([]fields.Version{1}, bodies.MaxVersionsPerOneHello+1)

	tests := []struct {
		name    string
		build   func() (frame.Frame, error)
		want    frame.Frame
		wantErr bool
	}{
		{
			name:  "empty",
			build: func() (frame.Frame, error) { return fb.NewHello(nil) },
			want:  frame.Frame{Body: bodies.Hello{}},
		},
		{
			name:  "client spec",
			build: func() (frame.Frame, error) { return fb.NewHello([]fields.Version{1, 2, 3}) },
			want:  frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1, 2, 3}}},
		},
		{
			name:  "server spec",
			build: func() (frame.Frame, error) { return fb.NewHello([]fields.Version{1, 4, 7, 11}) },
			want:  frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1, 4, 7, 11}}},
		},
		{
			name:  "deduplicates versions",
			build: func() (frame.Frame, error) { return fb.NewHello(tooManyOnes) },
			want:  frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1}}},
		},
		{
			name:  "drops version 0",
			build: func() (frame.Frame, error) { return fb.NewHello([]fields.Version{0, 1}) },
			want:  frame.Frame{Body: bodies.Hello{Versions: []fields.Version{1}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, e := tt.build()
			testutil.AssertErr(t, e, tt.wantErr)

			if tt.wantErr {
				return
			}

			testutil.AssertSameEncoding(t, got.Body, tt.want.Body)
		})
	}
}

func TestFrameBuilder_LimitExceeded(t *testing.T) {
	fb := NewFrameBuilder(1)

	if _, e := fb.NewHello([]fields.Version{1}); e == nil {
		t.Error("got nil err, want frame size error")
	}
}
