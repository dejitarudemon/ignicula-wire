package decoder

import (
	"errors"
	"fmt"

	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

const (
	resultNotOK = 0
	resultOK    = 1
)

// errorDecoder builds a protocol error from its traceback ID and the error details in c.
type errorDecoder func(tracebackID fields.TracebackID, c *cursor) (err.ProtocolError, error)

// errorDecoders maps every protocol error code to the decoder of its details.
var errorDecoders = map[fields.Error]errorDecoder{
	fields.NoHello: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorNoHelloWithTracebackID(id), nil
	},
	fields.CommandNotImplemented: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorCommandNotImplementedWithTracebackID(id), nil
	},
	fields.UnexpectedCommand: func(id fields.TracebackID, c *cursor) (err.ProtocolError, error) {
		expected, e := c.uint8()
		if e != nil {
			return nil, e
		}
		return errs.NewErrorUnexpectedCommandWithTracebackID(0, fields.Command(expected), id), nil
	},
	fields.UnsupportedCommand: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorUnsupportedCommandWithTracebackID(0, id), nil
	},
	fields.RequestsConflict: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorRequestsConflictWithTracebackID(0, id), nil
	},
	fields.UnsupportedCompression: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorUnsupportedCompressionWithTracebackID(0, id), nil
	},
	fields.BodyLimitIsExceeded: func(id fields.TracebackID, c *cursor) (err.ProtocolError, error) {
		limit, e := c.uint32()
		if e != nil {
			return nil, e
		}
		return errs.NewErrorBodyLimitIsExceededWithTracebackID(0, fields.BodyLimit(limit), id), nil
	},
	fields.MismatchedChecksum: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorMismatchedChecksumWithTracebackID(0, 0, id), nil
	},
	fields.InternalError: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorInternalErrorWithTracebackID(errors.New(""), id), nil
	},
	fields.MalformedValue: func(id fields.TracebackID, c *cursor) (err.ProtocolError, error) {
		return errs.NewErrorMalformedValueWithTracebackID(string(c.rest()), id), nil
	},
	fields.NotFound: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorNotFoundWithTracebackID(fields.Key{}, id), nil
	},
	fields.ProhibitedCompression: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorProhibitedCompressionWithTracebackID(0, 0, id), nil
	},
	fields.RequestInterrupted: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorRequestInterruptedWithTracebackID(0, id), nil
	},
	fields.BatchLimitIsExceeded: func(id fields.TracebackID, c *cursor) (err.ProtocolError, error) {
		limit, e := c.uint32()
		if e != nil {
			return nil, e
		}
		return errs.NewErrorBatchLimitIsExceededWithTracebackID(0, fields.BatchLimit(limit), id), nil
	},
	fields.UnexpectedCommandInBatch: func(id fields.TracebackID, c *cursor) (err.ProtocolError, error) {
		number, e := c.uint32()
		if e != nil {
			return nil, e
		}
		return errs.NewErrorUnexpectedCommandInBatchWithTracebackID(0, fields.RequestNumber(number), id), nil
	},
	fields.InvalidRequestID: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorInvalidRequestIDWithTracebackID(0, 0, id), nil
	},
	fields.Unauthorized: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorUnauthorizedWithTracebackID([]byte{}, id), nil
	},
	fields.RestrictedRequest: func(id fields.TracebackID, _ *cursor) (err.ProtocolError, error) {
		return errs.NewErrorRestrictedRequestWithTracebackID(fields.Key{}, []byte{}, []byte{}, 0, 0, id), nil
	},
}

// answer decodes an answer payload.
func (d Decoder) answer(c *cursor) (body.Answer, error) {
	result, e := c.uint8()
	if e != nil {
		return nil, e
	}

	switch result {
	case resultNotOK:
		return d.answerErr(c)
	case resultOK:
	default:
		return nil, errs.NewErrorMalformedValue(fmt.Sprintf("unknown result: %v", result))
	}

	originalCommand, e := c.uint8()
	if e != nil {
		return nil, e
	}

	switch fields.Command(originalCommand) {
	case fields.Ping:
		return bodies.PingAnswer{}, nil
	case fields.Write:
		return bodies.WriteAnswer{}, nil
	case fields.Delete:
		return bodies.DeleteAnswer{}, nil
	case fields.Read:
		return d.answerRead(c)
	case fields.Handshake:
		return d.answerHandshake(c)
	case fields.Batch:
		return d.answerBatch(c)
	case fields.Answer:
		return nil, errs.NewErrorMalformedValue("got answer for answer")
	}

	return nil, errs.NewErrorMalformedValue(fmt.Sprintf("unknown command in answer: %v", originalCommand))
}

// answerHandshake decodes a handshake answer payload.
func (d Decoder) answerHandshake(c *cursor) (bodies.HandshakeAnswer, error) {
	compressions, e := d.decodeCompressions(c)
	if e != nil {
		return bodies.HandshakeAnswer{}, e
	}

	versions, e := d.decodeVersions(c)
	if e != nil {
		return bodies.HandshakeAnswer{}, e
	}

	return bodies.NewHandshakeAnswer(compressions, versions), nil
}

// answerRead decodes a read answer payload.
func (d Decoder) answerRead(c *cursor) (bodies.ReadAnswer, error) {
	v, e := d.decodeTypedValue(c)
	if e != nil {
		return bodies.ReadAnswer{}, e
	}

	return bodies.ReadAnswer{Value: v}, nil
}

// answerErr decodes an error answer payload.
func (d Decoder) answerErr(c *cursor) (bodies.ErrorAnswer, error) {
	code, e := c.uint16()
	if e != nil {
		return bodies.ErrorAnswer{}, e
	}

	rawID, e := c.bytes(fields.TracebackIDFieldSize)
	if e != nil {
		return bodies.ErrorAnswer{}, e
	}

	decode, ok := errorDecoders[fields.Error(code)]
	if !ok {
		return bodies.ErrorAnswer{}, errs.NewErrorMalformedValue(fmt.Sprintf("unknown error code: %v", code))
	}

	pe, e := decode(fields.TracebackID(rawID), c)
	if e != nil {
		return bodies.ErrorAnswer{}, e
	}

	return bodies.ErrorAnswer{Err: pe}, nil
}

// answerBatch decodes a batch answer payload.
func (d Decoder) answerBatch(c *cursor) (bodies.BatchAnswer, error) {
	n, e := c.uint32()
	if e != nil {
		return nil, e
	}

	results := make(bodies.BatchAnswer, 0, min(int(n), c.remaining()/BatchResultMinBodySize))

	for range n {
		result, e := d.batchResult(c)
		if e != nil {
			return nil, e
		}

		results = append(results, result)
	}

	return results, nil
}

// batchResult decodes one numbered batch result.
func (d Decoder) batchResult(c *cursor) (bodies.Result, error) {
	number, e := c.uint32()
	if e != nil {
		return bodies.Result{}, e
	}

	nested, e := c.length()
	if e != nil {
		return bodies.Result{}, e
	}

	sub, e := c.sub(nested)
	if e != nil {
		return bodies.Result{}, e
	}

	b, e := d.answer(sub)
	if e != nil {
		return bodies.Result{}, e
	}

	if e := sub.expectEnd(); e != nil {
		return bodies.Result{}, e
	}

	return bodies.Result{
		Number: fields.RequestNumber(number),
		Body:   b,
	}, nil
}
