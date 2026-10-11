package decoder

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor/compressors"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
	"github.com/dejitarudemon/ignicula-wire/v1/internal/specs"
	"github.com/dejitarudemon/ignicula-wire/v1/internal/testutil"
	"github.com/dejitarudemon/ignicula-wire/v1/value"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

var (
	zstd, _ = compressors.NewZstd(1 << 12)
	s2, _   = compressors.NewS2(1 << 12)

	testHash = [32]byte{0x01, 0x02, 0x03, 31: 0xFF}
)

var roundTripFrames = []struct {
	name string
	f    frame.Frame
}{
	{"handshake empty", frame.Frame{RequestID: 0, Body: bodies.NewHandshake("", [32]byte{}, []fields.Compression{})}},
	{"handshake", frame.Frame{RequestID: 0, Body: bodies.NewHandshake("test", testHash, []fields.Compression{fields.Zstd})}},
	{"handshake duplicate compressions", frame.Frame{RequestID: 0, Body: bodies.NewHandshake("test", testHash, []fields.Compression{fields.Zstd, fields.Zstd})}},
	{"handshake unknown compression", frame.Frame{RequestID: 0, Body: bodies.NewHandshake("test", testHash, []fields.Compression{fields.Zstd, fields.Compression(23)})}},
	{"handshake answer empty", frame.Frame{RequestID: 0, Body: bodies.NewHandshakeAnswer([]fields.Compression{}, nil)}},
	{"handshake answer", frame.Frame{RequestID: 0, Body: bodies.NewHandshakeAnswer([]fields.Compression{fields.Zstd}, []fields.Version{1})}},
	{"handshake answer unknown compression", frame.Frame{RequestID: 0, Body: bodies.NewHandshakeAnswer([]fields.Compression{fields.Zstd, fields.Compression(23)}, []fields.Version{1})}},

	{"read empty key", frame.Frame{RequestID: 1, Body: bodies.Read("")}},
	{"read", frame.Frame{RequestID: 1, Body: bodies.Read("key")}},
	{"read answer empty bytes", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Bytes{}}}},
	{"read answer bytes", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Bytes{0xFF}}}},
	{"read answer zero int", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Int(0)}}},
	{"read answer int", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Int(1)}}},
	{"read answer empty untyped array", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.UntypedArray{}}}},
	{"read answer untyped array", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.UntypedArray{values.Bytes{0xFF}, values.Int(1)}}}},
	{"read answer typed array", frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.TypedArray{
		ElemType: fields.Bytes,
		Elems:    []value.V{values.Bytes{0xFF}, values.Bytes{0x00}},
	}}}},

	{"write empty key", frame.Frame{RequestID: 1, Body: bodies.Write{Key: fields.Key(""), Value: values.Bytes{}}}},
	{"write bytes", frame.Frame{RequestID: 2, Body: bodies.Write{Key: fields.Key("k"), Value: values.Bytes{0xFF}}}},
	{"write int", frame.Frame{RequestID: 3, Body: bodies.Write{Key: fields.Key("key"), Value: values.Int(1)}}},
	{"write empty untyped array", frame.Frame{RequestID: 4, Body: bodies.Write{Key: fields.Key("key "), Value: values.UntypedArray{}}}},
	{"write untyped array", frame.Frame{RequestID: 5, Body: bodies.Write{Key: fields.Key("key key"), Value: values.UntypedArray{values.Bytes{0xFF}, values.Int(1)}}}},
	{"write typed array", frame.Frame{RequestID: 6, Body: bodies.Write{Key: fields.Key("key yek"), Value: values.TypedArray{
		ElemType: fields.Bytes,
		Elems:    []value.V{values.Bytes{0xFF}, values.Bytes{0x00}},
	}}}},
	{"write answer", frame.Frame{RequestID: 1, Body: bodies.WriteAnswer{}}},

	{"delete empty key", frame.Frame{RequestID: 1, Body: bodies.Delete("")}},
	{"delete", frame.Frame{RequestID: 1, Body: bodies.Delete("key")}},
	{"delete answer", frame.Frame{RequestID: 1, Body: bodies.DeleteAnswer{}}},

	{"ping", frame.Frame{RequestID: 1, Body: bodies.Ping{}}},
	{"ping answer", frame.Frame{RequestID: 1, Body: bodies.PingAnswer{}}},

	{"internal error", frame.Frame{RequestID: 0, Body: bodies.ErrorAnswer{Err: errs.NewErrorInternalError(nil)}}},
	{"malformed value", frame.Frame{RequestID: 1, Body: bodies.ErrorAnswer{Err: errs.NewErrorMalformedValue("bad value")}}},
	{"batch limit is exceeded", frame.Frame{RequestID: 1, Body: bodies.ErrorAnswer{Err: errs.NewErrorBatchLimitIsExceeded(0, 1)}}},
	{"unexpected command", frame.Frame{RequestID: 1, Body: bodies.ErrorAnswer{Err: errs.NewErrorUnexpectedCommand(fields.Handshake, fields.Command(10))}}},

	{"batch without flags", frame.Frame{RequestID: 1, Body: bodies.Batch{}}},
	{"batch sequential execution", frame.Frame{RequestID: 1, Body: bodies.Batch{IsSequentialExecution: true}}},
	{"batch one answer", frame.Frame{RequestID: 1, Body: bodies.Batch{IsOneAnswer: true}}},
	{"batch interrupt after error", frame.Frame{RequestID: 1, Body: bodies.Batch{InterruptAfterError: true}}},
	{"batch all flags", frame.Frame{RequestID: 1, Body: bodies.Batch{IsSequentialExecution: true, IsOneAnswer: true, InterruptAfterError: true}}},
	{"batch one request", frame.Frame{RequestID: 1, Body: bodies.Batch{Requests: []bodies.Request{
		{Number: 0, Body: bodies.Read("key")},
	}}}},
	{"batch duplicate numbers", frame.Frame{RequestID: 1, Body: bodies.Batch{Requests: []bodies.Request{
		{Number: 0, Body: bodies.Read("key")},
		{Number: 0, Body: bodies.Read("key")},
	}}}},
	{"batch mixed requests", frame.Frame{RequestID: 1, Body: bodies.Batch{Requests: []bodies.Request{
		{Number: 0, Body: bodies.Read("key")},
		{Number: 1, Body: bodies.Delete("key")},
	}}}},

	{"batch answer empty", frame.Frame{RequestID: 1, Body: bodies.BatchAnswer(nil)}},
	{"batch answer duplicate numbers", frame.Frame{RequestID: 1, Body: bodies.BatchAnswer{
		{Number: 0, Body: bodies.WriteAnswer{}},
		{Number: 0, Body: bodies.DeleteAnswer{}},
	}}},
	{"batch answer mixed results", frame.Frame{RequestID: 1, Body: bodies.BatchAnswer{
		{Number: 0, Body: bodies.WriteAnswer{}},
		{Number: 1, Body: bodies.ReadAnswer{Value: values.Int(123)}},
	}}},
}

var roundTripCompressors = []struct {
	name       string
	compressor compressor.Compressor
}{
	{"none", nil},
	{"zstd", zstd},
	{"s2", s2},
}

// rawFrame builds a frame with a valid checksum around an arbitrary body.
func rawFrame(command fields.Command, compression fields.Compression, body []byte) []byte {
	headers := cat(frame.MagicBytes, []byte{0x01, byte(command)}, u32(1), []byte{byte(compression)}, u32(uint32(len(body))))

	return cat(headers, body, u32(uint32(fields.NewChecksumWithParts(headers, body))))
}

func encodeFrame(t testing.TB, f frame.Frame, c compressor.Compressor) []byte {
	t.Helper()

	buf := buffer.Slice{}
	buf.Preallocate(f.Size())

	if e := f.Encode(&buf, c); e != nil {
		t.Fatalf("Encode() = %v", e)
	}

	return buf.Bytes()
}

func decodeFrameBytes(d Decoder, data []byte) (frame.Frame, error) {
	return d.DecodeFrame(bufio.NewReader(bytes.NewReader(data)))
}

func assertSameFrame(t *testing.T, got, want frame.Frame) {
	t.Helper()

	if got.RequestID != want.RequestID {
		t.Errorf("RequestID = %v, want %v", got.RequestID, want.RequestID)
	}

	testutil.AssertSameEncoding(t, got.Body, want.Body)
}

// isA returns a matcher reporting whether an error chain contains a T.
func isA[T error]() func(error) bool {
	return func(e error) bool {
		var target T
		return errors.As(e, &target)
	}
}

type fakeCompressor struct {
	code       fields.Compression
	decompress func([]byte) ([]byte, error)
}

func (f fakeCompressor) Code() fields.Compression              { return f.code }
func (f fakeCompressor) Compress(buf []byte) ([]byte, error)   { return buf, nil }
func (f fakeCompressor) Decompress(buf []byte) ([]byte, error) { return f.decompress(buf) }

func TestDecoder_Specs(t *testing.T) {
	for _, tt := range specs.Frames {
		t.Run(tt.Name, func(t *testing.T) {
			d := NewDecoder(1024, nil)
			reader := bufio.NewReader(bytes.NewReader(tt.Encoded))

			version, e := d.DecodePreamble(reader)
			if e != nil {
				t.Fatalf("DecodePreamble() = %v", e)
			}

			if version != fields.CurrentVersion {
				t.Errorf("DecodePreamble() = %v, want %v", version, fields.CurrentVersion)
			}

			got, e := d.DecodeFrame(reader)
			if e != nil {
				t.Fatalf("DecodeFrame() = %v", e)
			}

			assertSameFrame(t, got, tt.Frame)
		})
	}
}

func TestDecoder_RoundTrip(t *testing.T) {
	d := NewDecoder(1024, []compressor.Compressor{zstd, s2})

	for _, c := range roundTripCompressors {
		for _, tt := range roundTripFrames {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				got, e := decodeFrameBytes(d, encodeFrame(t, tt.f, c.compressor))
				if e != nil {
					t.Fatalf("DecodeFrame() = %v", e)
				}

				assertSameFrame(t, got, tt.f)
			})
		}
	}
}

func TestDecoder_DecodePreamble(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    fields.Version
		wantErr bool
	}{
		{"empty", []byte{}, 0, true},
		{"one byte", []byte{0x0A}, 0, true},
		{"magic without version", []byte{0x0A, 0xDB}, 0, true},
		{"swapped magic", []byte{0xDB, 0x0A, 0x01}, 0, true},
		{"wrong magic", []byte{0x11, 0xFF, 0x01}, 0, true},
		{"version 0", []byte{0x0A, 0xDB, 0x00}, 0, false},
		{"version 1", []byte{0x0A, 0xDB, 0x01}, 1, false},
		{"version 255", []byte{0x0A, 0xDB, 0xFF}, 255, false},
		{"followed by data", []byte{0x0A, 0xDB, 0xFF, 0x01}, 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(1024, nil)

			got, e := d.DecodePreamble(bufio.NewReader(bytes.NewReader(tt.data)))
			testutil.AssertErr(t, e, tt.wantErr)

			if got != tt.want {
				t.Errorf("DecodePreamble() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecoder_DecodePreamble_DoesNotConsume(t *testing.T) {
	d := NewDecoder(1024, nil)
	data := specs.Frames[0].Encoded
	reader := bufio.NewReader(bytes.NewReader(data))

	if _, e := d.DecodePreamble(reader); e != nil {
		t.Fatalf("DecodePreamble() = %v", e)
	}

	if _, e := d.DecodeFrame(reader); e != nil {
		t.Errorf("DecodeFrame() after DecodePreamble() = %v", e)
	}
}

func TestDecoder_DecodeFrameErrs(t *testing.T) {
	sentinel := errors.New("corrupted stream")
	valid := rawFrame(fields.Ping, fields.None, nil)
	corrupted := bytes.Clone(valid)
	corrupted[len(corrupted)-1] ^= 0xFF
	readAnswer := cat([]byte{0x01, byte(fields.Read), byte(fields.String)}, u32(4), []byte("data"))

	tests := []struct {
		name        string
		limit       fields.BodyLimit
		compressors []compressor.Compressor
		data        []byte
		match       func(error) bool
	}{
		{"empty reader", 1024, nil, []byte{}, isEOF(io.EOF)},
		{"truncated headers", 1024, nil, valid[:5], isEOF(io.ErrUnexpectedEOF)},
		{"truncated body", 1024, nil, rawFrame(fields.Read, fields.None, []byte("key"))[:15], isEOF(io.ErrUnexpectedEOF)},
		{"truncated checksum", 1024, nil, valid[:len(valid)-1], isEOF(io.ErrUnexpectedEOF)},
		{"body limit is exceeded", 1, nil, rawFrame(fields.Answer, fields.None, []byte{0x01, 0x03}), isA[errs.ErrorBodyLimitIsExceeded]()},
		{"unsupported compression", 1024, nil, rawFrame(fields.Ping, fields.Zstd, nil), isA[errs.ErrorUnsupportedCompression]()},
		{"mismatched checksum", 1024, nil, corrupted, isA[errs.ErrorMismatchedChecksum]()},
		{"unsupported command", 1024, nil, rawFrame(fields.Command(0x7F), fields.None, nil), isA[errs.ErrorUnsupportedCommand]()},
		{"malformed body", 1024, nil, rawFrame(fields.Write, fields.None, u32(10)), isA[errs.ErrorMalformedValue]()},
		{"trailing data after ping", 1024, nil, rawFrame(fields.Ping, fields.None, []byte{0xC0, 0xFF, 0xEE}), isA[err.TrailledError]()},
		{"trailing data after read answer", 1024, nil, rawFrame(fields.Answer, fields.None, cat(readAnswer, []byte("virus"))), isA[err.TrailledError]()},
		{
			"compressor error",
			1024,
			[]compressor.Compressor{fakeCompressor{code: fields.Zstd, decompress: func([]byte) ([]byte, error) { return nil, sentinel }}},
			rawFrame(fields.Ping, fields.Zstd, []byte{0x01}),
			func(e error) bool { return isA[err.DecodeError]()(e) && errors.Is(e, sentinel) },
		},
		{
			"trailing data after decompression",
			1024,
			[]compressor.Compressor{fakeCompressor{code: fields.S2, decompress: func([]byte) ([]byte, error) { return []byte{0x00, 0x01, 0x02}, nil }}},
			rawFrame(fields.Ping, fields.S2, []byte{0xFF}),
			func(e error) bool { return errors.Is(e, err.NewTrailledError(3, 0)) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.limit, tt.compressors)

			_, e := decodeFrameBytes(d, tt.data)
			if !tt.match(e) {
				t.Errorf("DecodeFrame() = %v", e)
			}
		})
	}
}

// isEOF matches a decode error wrapping target.
func isEOF(target error) func(error) bool {
	return func(e error) bool {
		return isA[err.DecodeError]()(e) && errors.Is(e, target)
	}
}

func TestDecoder_DecodeFrame_BodyErrorKeepsRequestID(t *testing.T) {
	d := NewDecoder(1024, nil)

	f, e := decodeFrameBytes(d, rawFrame(fields.Write, fields.None, u32(10)))
	if e == nil {
		t.Fatal("DecodeFrame() = nil, want error")
	}

	if f.RequestID != 1 {
		t.Errorf("RequestID = %v, want 1", f.RequestID)
	}

	if f.Body != nil {
		t.Errorf("Body = %v, want nil", f.Body)
	}
}

func TestDecoder_ReaderErrs(t *testing.T) {
	sentinel := errors.New("broken pipe")

	tests := []struct {
		name   string
		decode func(d Decoder, r *bufio.Reader) error
	}{
		{"DecodePreamble", func(d Decoder, r *bufio.Reader) error { _, e := d.DecodePreamble(r); return e }},
		{"DecodeFrame", func(d Decoder, r *bufio.Reader) error { _, e := d.DecodeFrame(r); return e }},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/nil reader", func(t *testing.T) {
			if e := tt.decode(NewDecoder(1024, nil), nil); !isA[err.DecodeError]()(e) {
				t.Errorf("got %v, want DecodeError", e)
			}
		})

		t.Run(tt.name+"/failing reader", func(t *testing.T) {
			e := tt.decode(NewDecoder(1024, nil), bufio.NewReader(iotest.ErrReader(sentinel)))
			if !isA[err.DecodeError]()(e) || !errors.Is(e, sentinel) {
				t.Errorf("got %v, want DecodeError wrapping %v", e, sentinel)
			}
		})
	}
}

func TestDecoder_ZipBomb(t *testing.T) {
	bomb := frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.Bytes(make([]byte, 1<<13))}}

	tests := []struct {
		name       string
		compressor compressor.Compressor
	}{
		{"zstd", zstd},
		{"s2", s2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(1<<12, []compressor.Compressor{zstd, s2})

			if _, e := decodeFrameBytes(d, encodeFrame(t, bomb, tt.compressor)); e == nil {
				t.Error("DecodeFrame() = nil, want error")
			}
		})
	}
}

func TestNewDecoder_KeepsFirstCompressorPerCode(t *testing.T) {
	first := fakeCompressor{code: fields.Zstd, decompress: func([]byte) ([]byte, error) { return nil, nil }}
	second := fakeCompressor{code: fields.Zstd, decompress: func([]byte) ([]byte, error) { return nil, errors.New("second") }}
	d := NewDecoder(1024, []compressor.Compressor{first, second})

	if _, e := decodeFrameBytes(d, rawFrame(fields.Ping, fields.Zstd, []byte{0x01})); e != nil {
		t.Errorf("DecodeFrame() = %v, want the first compressor to be used", e)
	}
}

func FuzzDecoder_Preamble(f *testing.F) {
	f.Add([]byte{0x0A, 0xDB, 0x01})
	f.Fuzz(func(t *testing.T, a []byte) {
		_, _ = NewDecoder(1024, nil).DecodePreamble(bufio.NewReader(bytes.NewBuffer(a)))
	})
}

func FuzzDecoder_BySpecs(f *testing.F) {
	for _, tt := range specs.Frames {
		f.Add(tt.Encoded)
	}
	f.Fuzz(func(t *testing.T, a []byte) {
		_, _ = NewDecoder(1024, nil).DecodeFrame(bufio.NewReader(bytes.NewBuffer(a)))
	})
}

func FuzzDecoder_RoundTrip(f *testing.F) {
	for _, c := range roundTripCompressors {
		for _, tt := range roundTripFrames {
			f.Add(encodeFrame(f, tt.f, c.compressor))
		}
	}
	f.Fuzz(func(t *testing.T, a []byte) {
		_, _ = NewDecoder(1<<12, []compressor.Compressor{zstd, s2}).DecodeFrame(bufio.NewReader(bytes.NewBuffer(a)))
	})
}
