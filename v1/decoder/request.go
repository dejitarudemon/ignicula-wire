package decoder

import (
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

// ping returns an empty ping body.
func (d Decoder) ping(*cursor) (bodies.Ping, error) {
	return bodies.Ping{}, nil
}

// read decodes a read body, which is the whole remaining key.
func (d Decoder) read(c *cursor) (bodies.Read, error) {
	return bodies.Read(c.rest()), nil
}

// delete decodes a delete body, which is the whole remaining key.
func (d Decoder) delete(c *cursor) (bodies.Delete, error) {
	return bodies.Delete(c.rest()), nil
}

// handshake decodes a handshake body.
func (d Decoder) handshake(c *cursor) (bodies.Handshake, error) {
	login, e := d.decodeLengthPrefixed(c)
	if e != nil {
		return bodies.Handshake{}, e
	}

	hash, e := c.bytes(bodies.HashFieldSize)
	if e != nil {
		return bodies.Handshake{}, e
	}

	compressions, e := d.decodeCompressions(c)
	if e != nil {
		return bodies.Handshake{}, e
	}

	return bodies.NewHandshake(string(login), [bodies.HashFieldSize]byte(hash), compressions), nil
}

// decodeCompressions reads a one-byte count followed by that many compression codes.
func (d Decoder) decodeCompressions(c *cursor) ([]fields.Compression, error) {
	n, e := c.uint8()
	if e != nil {
		return nil, e
	}

	raw, e := c.bytes(int(n) * fields.CompressionFieldSize)
	if e != nil {
		return nil, e
	}

	compressions := make([]fields.Compression, 0, n)
	for _, b := range raw {
		compressions = append(compressions, fields.Compression(b))
	}

	return compressions, nil
}

// decodeVersions reads a one-byte count followed by that many protocol versions.
func (d Decoder) decodeVersions(c *cursor) ([]fields.Version, error) {
	n, e := c.uint8()
	if e != nil {
		return nil, e
	}

	raw, e := c.bytes(int(n) * fields.VersionFieldSize)
	if e != nil {
		return nil, e
	}

	versions := make([]fields.Version, 0, n)
	for _, b := range raw {
		versions = append(versions, fields.Version(b))
	}

	return versions, nil
}

// write decodes a write body.
func (d Decoder) write(c *cursor) (bodies.Write, error) {
	key, e := d.decodeLengthPrefixed(c)
	if e != nil {
		return bodies.Write{}, e
	}

	v, e := d.decodeTypedValue(c)
	if e != nil {
		return bodies.Write{}, e
	}

	return bodies.Write{
		Key:   fields.Key(key),
		Value: v,
	}, nil
}

// batch decodes a batch body.
func (d Decoder) batch(c *cursor) (bodies.Batch, error) {
	flags, e := c.uint8()
	if e != nil {
		return bodies.Batch{}, e
	}

	n, e := c.uint32()
	if e != nil {
		return bodies.Batch{}, e
	}

	requests := make([]bodies.Request, 0, min(int(n), c.remaining()/BatchRequestMinBodySize))

	for range n {
		request, e := d.batchRequest(c)
		if e != nil {
			return bodies.Batch{}, e
		}

		requests = append(requests, request)
	}

	return bodies.Batch{
		Requests:              requests,
		IsSequentialExecution: flags&bodies.IsSequentialExecution != 0,
		InterruptAfterError:   flags&bodies.InterruptAfterError != 0,
		IsOneAnswer:           flags&bodies.IsOneAnswer != 0,
	}, nil
}

// batchRequest decodes one numbered batch request.
func (d Decoder) batchRequest(c *cursor) (bodies.Request, error) {
	number, e := c.uint32()
	if e != nil {
		return bodies.Request{}, e
	}

	requestNumber := fields.RequestNumber(number)

	command, e := c.uint8()
	if e != nil {
		return bodies.Request{}, e
	}

	if fields.Command(command) == fields.Batch {
		return bodies.Request{}, errs.NewErrorUnexpectedCommandInBatch(fields.Command(command), requestNumber)
	}

	nested, e := c.length()
	if e != nil {
		return bodies.Request{}, e
	}

	sub, e := c.sub(nested)
	if e != nil {
		return bodies.Request{}, e
	}

	b, e := d.decodeBody(sub, fields.Command(command))
	if e != nil {
		return bodies.Request{}, e
	}

	if e := sub.expectEnd(); e != nil {
		return bodies.Request{}, e
	}

	return bodies.Request{
		Number: requestNumber,
		Body:   b,
	}, nil
}
