package builder

import (
	"fmt"

	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
	"github.com/dejitarudemon/ignicula-wire/v1/value"
)

const (
	// handshakeRequestID is the request ID used by handshake frames.
	handshakeRequestID = fields.RequestID(0)
)

// FrameBuilder builds protocol frames and rejects those larger than a size limit.
type FrameBuilder struct {
	limit int
}

// NewFrameBuilder returns a FrameBuilder that rejects frames whose encoded size exceeds limit bytes.
func NewFrameBuilder(limit int) FrameBuilder {
	return FrameBuilder{
		limit: limit,
	}
}

// requestIDIsNotZero reports an error when requestID is zero.
func (fb FrameBuilder) requestIDIsNotZero(requestID fields.RequestID) error {
	if requestID == 0 {
		return err.NewBuildError("request ID is 0", nil)
	}

	return nil
}

// frameSizeLowerLimit reports an error when size exceeds the builder limit.
func (fb FrameBuilder) frameSizeLowerLimit(size int) error {
	if size > fb.limit {
		return err.NewBuildError(fmt.Sprintf("frame size is %v, but limit is %v", size, fb.limit), nil)
	}

	return nil
}

// NewHandshake returns a handshake frame with request ID 0.
// It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewHandshake(login string, hash [32]byte, compressions []fields.Compression) (frame.Frame, error) {
	f := frame.Frame{
		RequestID: handshakeRequestID,
		Body:      bodies.NewHandshake(login, hash, compressions),
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewHandshakeAnswer returns a handshake answer frame with request ID 0.
// compressions are algorithms the server accepts; versions are protocol versions
// granted for use without re-authentication.
// It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewHandshakeAnswer(compressions []fields.Compression, versions []fields.Version) (frame.Frame, error) {
	f := frame.Frame{
		RequestID: handshakeRequestID,
		Body:      bodies.NewHandshakeAnswer(compressions, versions),
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewRead returns a read frame for key.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewRead(requestID fields.RequestID, key fields.Key) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.Read(key),
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewReadAnswer returns a read answer frame carrying value.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewReadAnswer(requestID fields.RequestID, value value.V) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.ReadAnswer{Value: value},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewWrite returns a write frame storing value under key.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewWrite(requestID fields.RequestID, key fields.Key, value value.V) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.Write{Key: key, Value: value},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewWriteAnswer returns a successful write answer frame.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewWriteAnswer(requestID fields.RequestID) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.WriteAnswer{},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewDelete returns a delete frame for key.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewDelete(requestID fields.RequestID, key fields.Key) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.Delete(key),
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewDeleteAnswer returns a successful delete answer frame.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewDeleteAnswer(requestID fields.RequestID) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.DeleteAnswer{},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewPing returns a ping frame.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewPing(requestID fields.RequestID) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.Ping{},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewPingAnswer returns a ping answer frame.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewPingAnswer(requestID fields.RequestID) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.PingAnswer{},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewErrAnswer returns an error answer frame for pe.
// It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewErrAnswer(requestID fields.RequestID, pe err.ProtocolError) (frame.Frame, error) {
	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.ErrorAnswer{Err: pe},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewBatch returns a batch frame built from batch.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewBatch(requestID fields.RequestID, batch BatchRequestsBuilder) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	requests := make([]bodies.Request, len(batch.requests))
	copy(requests, batch.requests)

	f := frame.Frame{
		RequestID: requestID,
		Body: bodies.Batch{
			IsSequentialExecution: batch.isSequentialExecution,
			IsOneAnswer:           batch.isOneAnswer,
			InterruptAfterError:   batch.interruptAfterError,

			Requests: requests,
		},
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}

// NewBatchAnswer returns a batch answer frame built from batch.
// requestID must be non-zero. It returns an error when the body is invalid or the encoded frame exceeds the size limit.
func (fb FrameBuilder) NewBatchAnswer(requestID fields.RequestID, batch BatchResultsBuilder) (frame.Frame, error) {
	if err := fb.requestIDIsNotZero(requestID); err != nil {
		return frame.Frame{}, err
	}

	results := make([]bodies.Result, len(batch.results))
	copy(results, batch.results)

	f := frame.Frame{
		RequestID: requestID,
		Body:      bodies.BatchAnswer(results),
	}

	if err := f.IsValid(); err != nil {
		return frame.Frame{}, err
	}

	return f, fb.frameSizeLowerLimit(f.Size())
}
