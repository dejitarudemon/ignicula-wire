package builder

import (
	"math"
	"slices"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
	"github.com/dejitarudemon/ignicula-wire/v1/internal/testutil"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func TestFrameBuilder(t *testing.T) {
	fb := NewFrameBuilder(1 << 10)
	internalErr := errs.NewErrorInternalErrorWithTracebackID(nil, fields.TracebackID{})
	tooManyZstd := slices.Repeat([]fields.Compression{fields.Zstd}, int(bodies.MaxCompressionsPerOneHandshake)+1)
	almostTooManyZstd := slices.Repeat([]fields.Compression{fields.Zstd}, int(bodies.MaxCompressionsPerOneHandshake)-1)

	tests := []struct {
		name    string
		build   func() (frame.Frame, error)
		want    frame.Frame
		wantErr bool
	}{
		{
			name:  "handshake empty",
			build: func() (frame.Frame, error) { return fb.NewHandshake("", [32]byte{}, nil) },
			want:  frame.Frame{RequestID: 0, Body: bodies.Handshake{}},
		},
		{
			name:  "handshake with login",
			build: func() (frame.Frame, error) { return fb.NewHandshake("user", [32]byte{}, nil) },
			want:  frame.Frame{RequestID: 0, Body: bodies.Handshake{Login: "user"}},
		},
		{
			name:  "handshake deduplicates compressions",
			build: func() (frame.Frame, error) { return fb.NewHandshake("", [32]byte{}, tooManyZstd) },
			want:  frame.Frame{RequestID: 0, Body: bodies.Handshake{Compressions: []fields.Compression{fields.Zstd}}},
		},
		{
			name:  "handshake answer empty",
			build: func() (frame.Frame, error) { return fb.NewHandshakeAnswer(nil, nil) },
			want:  frame.Frame{RequestID: 0, Body: bodies.HandshakeAnswer{}},
		},
		{
			name:  "handshake answer deduplicates compressions",
			build: func() (frame.Frame, error) { return fb.NewHandshakeAnswer(tooManyZstd, nil) },
			want:  frame.Frame{RequestID: 0, Body: bodies.HandshakeAnswer{Compressions: []fields.Compression{fields.Zstd}}},
		},
		{
			name:  "handshake answer deduplicates few compressions",
			build: func() (frame.Frame, error) { return fb.NewHandshakeAnswer(almostTooManyZstd, []fields.Version{1, 1}) },
			want: frame.Frame{RequestID: 0, Body: bodies.HandshakeAnswer{
				Compressions: []fields.Compression{fields.Zstd},
				Versions:     []fields.Version{1},
			}},
		},

		{
			name:    "ping zero request id",
			build:   func() (frame.Frame, error) { return fb.NewPing(0) },
			wantErr: true,
		},
		{
			name:  "ping",
			build: func() (frame.Frame, error) { return fb.NewPing(1) },
			want:  frame.Frame{RequestID: 1, Body: bodies.Ping{}},
		},
		{
			name:  "ping max request id",
			build: func() (frame.Frame, error) { return fb.NewPing(math.MaxUint32) },
			want:  frame.Frame{RequestID: math.MaxUint32, Body: bodies.Ping{}},
		},
		{
			name:    "ping answer zero request id",
			build:   func() (frame.Frame, error) { return fb.NewPingAnswer(0) },
			wantErr: true,
		},
		{
			name:  "ping answer",
			build: func() (frame.Frame, error) { return fb.NewPingAnswer(1) },
			want:  frame.Frame{RequestID: 1, Body: bodies.PingAnswer{}},
		},

		{
			name:    "read zero request id and empty key",
			build:   func() (frame.Frame, error) { return fb.NewRead(0, fields.Key("")) },
			wantErr: true,
		},
		{
			name:    "read zero request id",
			build:   func() (frame.Frame, error) { return fb.NewRead(0, fields.Key("key")) },
			wantErr: true,
		},
		{
			name:    "read empty key",
			build:   func() (frame.Frame, error) { return fb.NewRead(1, fields.Key("")) },
			wantErr: true,
		},
		{
			name:  "read",
			build: func() (frame.Frame, error) { return fb.NewRead(1, fields.Key("key")) },
			want:  frame.Frame{RequestID: 1, Body: bodies.Read("key")},
		},
		{
			name:    "read answer zero request id",
			build:   func() (frame.Frame, error) { return fb.NewReadAnswer(0, nil) },
			wantErr: true,
		},
		{
			name:    "read answer nil value",
			build:   func() (frame.Frame, error) { return fb.NewReadAnswer(1, nil) },
			wantErr: true,
		},
		{
			name:    "read answer invalid json",
			build:   func() (frame.Frame, error) { return fb.NewReadAnswer(1, values.JSON("-")) },
			wantErr: true,
		},
		{
			name:  "read answer",
			build: func() (frame.Frame, error) { return fb.NewReadAnswer(1, values.Int(3)) },
			want:  frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Int(3)}},
		},

		{
			name:    "write zero request id",
			build:   func() (frame.Frame, error) { return fb.NewWrite(0, fields.Key(""), nil) },
			wantErr: true,
		},
		{
			name:    "write empty key",
			build:   func() (frame.Frame, error) { return fb.NewWrite(1, fields.Key(""), nil) },
			wantErr: true,
		},
		{
			name:    "write nil value",
			build:   func() (frame.Frame, error) { return fb.NewWrite(1, fields.Key("key"), nil) },
			wantErr: true,
		},
		{
			name:    "write invalid json",
			build:   func() (frame.Frame, error) { return fb.NewWrite(1, fields.Key("key"), values.JSON("-")) },
			wantErr: true,
		},
		{
			name:  "write",
			build: func() (frame.Frame, error) { return fb.NewWrite(1, fields.Key("key"), values.Int(3)) },
			want:  frame.Frame{RequestID: 1, Body: bodies.Write{Key: fields.Key("key"), Value: values.Int(3)}},
		},
		{
			name:    "write answer zero request id",
			build:   func() (frame.Frame, error) { return fb.NewWriteAnswer(0) },
			wantErr: true,
		},
		{
			name:  "write answer",
			build: func() (frame.Frame, error) { return fb.NewWriteAnswer(1) },
			want:  frame.Frame{RequestID: 1, Body: bodies.WriteAnswer{}},
		},

		{
			name:    "delete zero request id and empty key",
			build:   func() (frame.Frame, error) { return fb.NewDelete(0, fields.Key("")) },
			wantErr: true,
		},
		{
			name:    "delete zero request id",
			build:   func() (frame.Frame, error) { return fb.NewDelete(0, fields.Key("key")) },
			wantErr: true,
		},
		{
			name:    "delete empty key",
			build:   func() (frame.Frame, error) { return fb.NewDelete(1, fields.Key("")) },
			wantErr: true,
		},
		{
			name:  "delete",
			build: func() (frame.Frame, error) { return fb.NewDelete(1, fields.Key("key")) },
			want:  frame.Frame{RequestID: 1, Body: bodies.Delete("key")},
		},
		{
			name:    "delete answer zero request id",
			build:   func() (frame.Frame, error) { return fb.NewDeleteAnswer(0) },
			wantErr: true,
		},
		{
			name:  "delete answer",
			build: func() (frame.Frame, error) { return fb.NewDeleteAnswer(1) },
			want:  frame.Frame{RequestID: 1, Body: bodies.DeleteAnswer{}},
		},

		{
			name:    "error answer nil error",
			build:   func() (frame.Frame, error) { return fb.NewErrAnswer(0, nil) },
			wantErr: true,
		},
		{
			name:    "error answer nil error with request id",
			build:   func() (frame.Frame, error) { return fb.NewErrAnswer(1, nil) },
			wantErr: true,
		},
		{
			name:  "error answer zero request id",
			build: func() (frame.Frame, error) { return fb.NewErrAnswer(0, internalErr) },
			want:  frame.Frame{RequestID: 0, Body: bodies.ErrorAnswer{Err: internalErr}},
		},
		{
			name:  "error answer",
			build: func() (frame.Frame, error) { return fb.NewErrAnswer(1, internalErr) },
			want:  frame.Frame{RequestID: 1, Body: bodies.ErrorAnswer{Err: internalErr}},
		},

		{
			name:    "batch zero request id and no requests",
			build:   func() (frame.Frame, error) { return fb.NewBatch(0, *NewBatchRequestsBuilder()) },
			wantErr: true,
		},
		{
			name:    "batch no requests",
			build:   func() (frame.Frame, error) { return fb.NewBatch(1, *NewBatchRequestsBuilder()) },
			wantErr: true,
		},
		{
			name: "batch zero request id",
			build: func() (frame.Frame, error) {
				return fb.NewBatch(0, *NewBatchRequestsBuilder().AddRead(fields.Key("qwerty")))
			},
			wantErr: true,
		},
		{
			name:    "batch invalid request",
			build:   func() (frame.Frame, error) { return fb.NewBatch(1, *NewBatchRequestsBuilder().AddRead(fields.Key(""))) },
			wantErr: true,
		},
		{
			name: "batch",
			build: func() (frame.Frame, error) {
				return fb.NewBatch(1, *NewBatchRequestsBuilder().OneAnswer(true).AddRead(fields.Key("qwerty")))
			},
			want: frame.Frame{RequestID: 1, Body: bodies.Batch{
				IsOneAnswer: true,
				Requests:    []bodies.Request{{Number: 0, Body: bodies.Read("qwerty")}},
			}},
		},
		{
			name:    "batch answer zero request id and no results",
			build:   func() (frame.Frame, error) { return fb.NewBatchAnswer(0, *NewBatchResultsBuilder()) },
			wantErr: true,
		},
		{
			name:    "batch answer no results",
			build:   func() (frame.Frame, error) { return fb.NewBatchAnswer(1, *NewBatchResultsBuilder()) },
			wantErr: true,
		},
		{
			name:    "batch answer zero request id",
			build:   func() (frame.Frame, error) { return fb.NewBatchAnswer(0, *NewBatchResultsBuilder().AddWrite(0)) },
			wantErr: true,
		},
		{
			name: "batch answer invalid result",
			build: func() (frame.Frame, error) {
				return fb.NewBatchAnswer(1, *NewBatchResultsBuilder().AddRead(1, values.JSON("qwerty")))
			},
			wantErr: true,
		},
		{
			name: "batch answer",
			build: func() (frame.Frame, error) {
				return fb.NewBatchAnswer(1, *NewBatchResultsBuilder().AddRead(1, values.Int(123)))
			},
			want: frame.Frame{RequestID: 1, Body: bodies.BatchAnswer{{Number: 1, Body: bodies.ReadAnswer{Value: values.Int(123)}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, e := tt.build()
			testutil.AssertErr(t, e, tt.wantErr)

			if tt.wantErr {
				return
			}

			if got.RequestID != tt.want.RequestID {
				t.Errorf("RequestID = %v, want %v", got.RequestID, tt.want.RequestID)
			}

			testutil.AssertSameEncoding(t, got.Body, tt.want.Body)
		})
	}
}

func TestFrameBuilder_LimitExceeded(t *testing.T) {
	fb := NewFrameBuilder(1)
	internalErr := errs.NewErrorInternalErrorWithTracebackID(nil, fields.TracebackID{})

	tests := []struct {
		name  string
		build func() (frame.Frame, error)
	}{
		{"handshake", func() (frame.Frame, error) { return fb.NewHandshake("", [32]byte{}, nil) }},
		{"handshake answer", func() (frame.Frame, error) { return fb.NewHandshakeAnswer(nil, nil) }},
		{"ping", func() (frame.Frame, error) { return fb.NewPing(1) }},
		{"ping answer", func() (frame.Frame, error) { return fb.NewPingAnswer(1) }},
		{"read", func() (frame.Frame, error) { return fb.NewRead(1, fields.Key("key")) }},
		{"read answer", func() (frame.Frame, error) { return fb.NewReadAnswer(1, values.Int(1)) }},
		{"write", func() (frame.Frame, error) { return fb.NewWrite(1, fields.Key("key"), values.Int(1)) }},
		{"write answer", func() (frame.Frame, error) { return fb.NewWriteAnswer(1) }},
		{"delete", func() (frame.Frame, error) { return fb.NewDelete(1, fields.Key("key")) }},
		{"delete answer", func() (frame.Frame, error) { return fb.NewDeleteAnswer(1) }},
		{"error answer", func() (frame.Frame, error) { return fb.NewErrAnswer(1, internalErr) }},
		{"batch", func() (frame.Frame, error) {
			return fb.NewBatch(1, *NewBatchRequestsBuilder().AddRead(fields.Key("key")))
		}},
		{"batch answer", func() (frame.Frame, error) { return fb.NewBatchAnswer(1, *NewBatchResultsBuilder().AddWrite(0)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, e := tt.build(); e == nil {
				t.Error("got nil err, want frame size error")
			}
		})
	}
}

func TestFrameBuilder_BatchIsCopied(t *testing.T) {
	fb := NewFrameBuilder(1 << 10)
	b := NewBatchRequestsBuilder().AddRead(fields.Key("key"))

	f, e := fb.NewBatch(1, *b)
	if e != nil {
		t.Fatalf("NewBatch() = %v", e)
	}

	b.requests[0].Body = bodies.Read("changed")

	testutil.AssertSameEncoding(t, f.Body, bodies.Batch{Requests: []bodies.Request{{Number: 0, Body: bodies.Read("key")}}})
}
