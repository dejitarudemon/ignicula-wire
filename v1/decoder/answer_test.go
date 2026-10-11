package decoder

import (
	"errors"
	"math"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

var testTracebackID = fields.TracebackID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

func allProtocolErrors() []err.ProtocolError {
	id := testTracebackID

	return []err.ProtocolError{
		errs.NewErrorNoHelloWithTracebackID(id),
		errs.NewErrorCommandNotImplementedWithTracebackID(id),
		errs.NewErrorUnexpectedCommandWithTracebackID(fields.Batch, fields.Handshake, id),
		errs.NewErrorUnsupportedCommandWithTracebackID(9, id),
		errs.NewErrorRequestsConflictWithTracebackID(3, id),
		errs.NewErrorUnsupportedCompressionWithTracebackID(7, id),
		errs.NewErrorBodyLimitIsExceededWithTracebackID(100, 50, id),
		errs.NewErrorMismatchedChecksumWithTracebackID(1, 2, id),
		errs.NewErrorInternalErrorWithTracebackID(errors.New("boom"), id),
		errs.NewErrorMalformedValueWithTracebackID("bad json", id),
		errs.NewErrorNotFoundWithTracebackID(fields.Key("k"), id),
		errs.NewErrorProhibitedCompressionWithTracebackID(fields.Zstd, fields.Ping, id),
		errs.NewErrorRequestInterruptedWithTracebackID(4, id),
		errs.NewErrorBatchLimitIsExceededWithTracebackID(10, 8, id),
		errs.NewErrorUnexpectedCommandInBatchWithTracebackID(fields.Handshake, 2, id),
		errs.NewErrorInvalidRequestIDWithTracebackID(fields.Read, 0, id),
		errs.NewErrorUnauthorizedWithTracebackID([]byte("src"), id),
		errs.NewErrorRestrictedRequestWithTracebackID(fields.Key("id"), fields.Key("src"), fields.Key("key"), fields.Read, 1, id),
	}
}

func TestDecoder_answerErr_AllCodes(t *testing.T) {
	protocolErrors := allProtocolErrors()

	if len(protocolErrors) != len(errorDecoders) {
		t.Fatalf("test covers %v codes, decoder knows %v", len(protocolErrors), len(errorDecoders))
	}

	for _, pe := range protocolErrors {
		t.Run(pe.Code().String(), func(t *testing.T) {
			want := bodies.ErrorAnswer{Err: pe}
			c := newCursor(encodeBody(t, want))
			got, e := NewDecoder(1024, nil).answer(c)
			assertDecoded(t, got, e, c, want)
		})
	}
}

func TestDecoder_answerErr_MalformedValueMessage(t *testing.T) {
	for _, msg := range []string{"", "some message"} {
		name := "empty"
		if msg != "" {
			name = "non-empty"
		}

		t.Run(name, func(t *testing.T) {
			pe := errs.NewErrorMalformedValueWithTracebackID(msg, testTracebackID)
			c := newCursor(encodeBody(t, bodies.ErrorAnswer{Err: pe}))

			got, e := NewDecoder(1024, nil).answer(c)
			if e != nil {
				t.Fatalf("answer() = %v", e)
			}

			if c.remaining() != 0 {
				t.Fatalf("answer() left %v bytes unread", c.remaining())
			}

			if got.(bodies.ErrorAnswer).Err.Error() != pe.Error() {
				t.Errorf("Error() = %q, want %q", got.(bodies.ErrorAnswer).Err.Error(), pe.Error())
			}
		})
	}
}

func TestDecoder_answerErr_Errs(t *testing.T) {
	id := testTracebackID[:]
	head := func(code fields.Error) []byte {
		return cat([]byte{resultNotOK}, u16(uint16(code)), id)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"without code", []byte{resultNotOK, 0x00}},
		{"without traceback id", cat([]byte{resultNotOK}, u16(0), id[:15])},
		{"unknown code", head(fields.RestrictedRequest + 1)},
		{"unknown code max", head(math.MaxUint16)},
		{"unexpected command without expected", head(fields.UnexpectedCommand)},
		{"body limit without limit", cat(head(fields.BodyLimitIsExceeded), []byte{0x00, 0x00, 0x00})},
		{"batch limit without limit", cat(head(fields.BatchLimitIsExceeded), []byte{0x00})},
		{"unexpected command in batch without number", head(fields.UnexpectedCommandInBatch)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, e := NewDecoder(1024, nil).answer(newCursor(tt.data))
			assertMalformed(t, e)
		})
	}
}

func TestDecoder_answer_OK(t *testing.T) {
	tests := []struct {
		name string
		want body.Answer
	}{
		{"ping", bodies.PingAnswer{}},
		{"write", bodies.WriteAnswer{}},
		{"delete", bodies.DeleteAnswer{}},
		{"read", bodies.ReadAnswer{Value: values.JSON(`{}`)}},
		{"handshake", bodies.NewHandshakeAnswer([]fields.Compression{fields.S2}, []fields.Version{1})},
		{"handshake without compressions", bodies.NewHandshakeAnswer(nil, nil)},
		{"batch empty", bodies.BatchAnswer{}},
		{"batch", bodies.BatchAnswer{
			{Number: 0, Body: bodies.ReadAnswer{Value: values.Uint(1)}},
			{Number: 1, Body: bodies.ErrorAnswer{Err: errs.NewErrorMalformedValueWithTracebackID("bad", testTracebackID)}},
			{Number: 2, Body: bodies.ErrorAnswer{Err: errs.NewErrorNotFoundWithTracebackID(fields.Key("k"), testTracebackID)}},
			{Number: 3, Body: bodies.DeleteAnswer{}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCursor(encodeBody(t, tt.want))
			got, e := NewDecoder(1024, nil).answer(c)
			assertDecoded(t, got, e, c, tt.want)
		})
	}
}

func TestDecoder_answer_OKErrs(t *testing.T) {
	ok := func(command fields.Command) []byte {
		return []byte{resultOK, byte(command)}
	}
	pingResult := cat(u32(0), u32(2), ok(fields.Ping))

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"unknown result", []byte{0x02}},
		{"answer for answer", ok(fields.Answer)},
		{"unknown command", []byte{resultOK, 0xFF}},
		{"handshake without compressions len", ok(fields.Handshake)},
		{"handshake compressions shorter than len", cat(ok(fields.Handshake), []byte{2, 1})},
		{"read without type", ok(fields.Read)},
		{"read unknown type", cat(ok(fields.Read), []byte{0xFF})},
		{"read short value", cat(ok(fields.Read), []byte{byte(fields.Uint)}, u32(0))},
		{"batch without len", cat(ok(fields.Batch), []byte{0x00})},
		{"batch missing result", cat(ok(fields.Batch), u32(2), pingResult)},
		{"batch result without number", cat(ok(fields.Batch), u32(1), []byte{0x00})},
		{"batch result without body len", cat(ok(fields.Batch), u32(1), u32(0))},
		{"batch result body longer than batch", cat(ok(fields.Batch), u32(1), u32(0), u32(100), ok(fields.Ping))},
		{"batch result body len overflows", cat(ok(fields.Batch), u32(1), u32(0), u32(math.MaxUint32))},
		{"batch result malformed body", cat(ok(fields.Batch), u32(1), u32(0), u32(1), []byte{resultOK})},
		{"batch result trailing bytes", cat(ok(fields.Batch), u32(1), u32(0), u32(3), ok(fields.Ping), []byte{0x00})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, e := NewDecoder(1024, nil).answer(newCursor(tt.data))
			assertMalformed(t, e)
		})
	}
}
