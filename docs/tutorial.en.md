Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Tutorials

Step-by-step frame examples off the network. A working TCP server is in the [README](../README.md#example-tcp-server). How the protocol is structured is in [how-it-works](how-it-works.en.md).

The snippets encode into memory. On a socket use `bufio.NewReader(conn)` instead of `bytes.NewReader`.

## 1. Agree on a version (Hello)

Client and server exchange version-0 frames. Working versions are the intersection of the two lists. `NewHello` drops version 0, duplicates, and values greater than 255.

```go
package main

import (
	"bufio"
	"bytes"
	"fmt"

	v0bodies "github.com/dejitarudemon/ignicula-wire/v0/body/bodies"
	v0builder "github.com/dejitarudemon/ignicula-wire/v0/builder"
	v0decoder "github.com/dejitarudemon/ignicula-wire/v0/decoder"
	v0fields "github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

func main() {
	client := v0builder.NewFrameBuilder(1 << 10)
	hello, err := client.NewHello([]v0fields.Version{1, 2, 3})
	if err != nil {
		panic(err)
	}

	var out buffer.Slice
	out.Preallocate(hello.Size())
	if err := hello.Encode(&out); err != nil {
		panic(err)
	}

	d := v0decoder.NewDecoder()
	r := bufio.NewReader(bytes.NewReader(out.Bytes()))

	version, err := d.DecodePreamble(r)
	if err != nil {
		panic(err)
	}
	fmt.Println("preamble version:", version) // 0

	got, err := d.DecodeFrame(r)
	if err != nil {
		panic(err)
	}

	mine := hello.Body.(v0bodies.Hello)
	peer := got.Body.(v0bodies.Hello)
	fmt.Println("common:", mine.Common(peer))
}
```

By hand, without the builder (import `v0/frame`). `NewHello` drops 0, duplicates, and values greater than 255; a `Hello` literal does not.

```go
hello := v0frame.Frame{
	Body: v0bodies.Hello{Versions: []v0fields.Version{1, 2, 3}},
}
if err := hello.IsValid(); err != nil {
	panic(err)
}

var out buffer.Slice
out.Preallocate(hello.Size())
if err := hello.Encode(&out); err != nil {
	panic(err)
}
```

Fields on the wire — magic, version, Version Len, versions, CRC-32/XFER:

```go
body := v0bodies.Hello{Versions: []v0fields.Version{1, 2, 3}}

var out buffer.Slice
out.Append(v0frame.MagicBytes)
v0fields.Version(0).Encode(&out)
v0fields.VersionLen(body.Size()).Encode(&out)
body.Encode(&out)
v0fields.NewChecksum(out.Bytes()).Encode(&out)
```

`DecodePreamble` does not consume bytes: use `v0/decoder` for version `0` and `v1/decoder` for version `1`. If the intersection is empty, the client closes the connection.

Specification example:

```text
Client → server:  0A DB 00 03 01 02 03 6E 38 99 00
Server → client:  0A DB 00 04 01 04 07 0B 3B 78 5D 98
Common versions: 1
```

## 2. Handshake

After Hello the client sends a version-1 frame with `RequestID = 0`. Compression is forbidden on Handshake. The hash is Argon2id of the password; use zeros for anonymous login.

```go
package main

import (
	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func main() {
	fb := builder.NewFrameBuilder(1 << 20)
	var hash [32]byte

	hs, err := fb.NewHandshake("user", hash, []fields.Compression{fields.Zstd, fields.S2})
	if err != nil {
		panic(err)
	}

	var out buffer.Slice
	out.Preallocate(hs.Size())
	if err := hs.Encode(&out, nil); err != nil {
		panic(err)
	}

	_ = out.Bytes()
}
```

By hand (imports `v1/frame` and `v1/body/bodies`):

```go
var hash [32]byte
hs := frame.Frame{
	RequestID: 0,
	Body:      bodies.NewHandshake("user", hash, []fields.Compression{fields.Zstd, fields.S2}),
}
if err := hs.IsValid(); err != nil {
	panic(err)
}

var out buffer.Slice
out.Preallocate(hs.Size())
if err := hs.Encode(&out, nil); err != nil {
	panic(err)
}
```

Build the server reply with `NewHandshakeAnswer(compressions, versions)` or `frame.Frame{RequestID: 0, Body: bodies.NewHandshakeAnswer(nil, []fields.Version{1})}`. `versions` are protocol versions granted for use without re-authentication. Later requests must use a non-zero `RequestID`.

## 3. Write and read

```go
package main

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/decoder"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func main() {
	fb := builder.NewFrameBuilder(1 << 20)

	write, err := fb.NewWrite(2, fields.Key("code"), values.String("IDDQD"))
	if err != nil {
		panic(err)
	}

	var out buffer.Slice
	out.Preallocate(write.Size())
	if err := write.Encode(&out, nil); err != nil {
		panic(err)
	}

	d := decoder.NewDecoder(1<<20, nil)
	got, err := d.DecodeFrame(bufio.NewReader(bytes.NewReader(out.Bytes())))
	if err != nil {
		panic(err)
	}
	fmt.Println(got.RequestID, got.Body)

	read, err := fb.NewRead(1, fields.Key("code"))
	if err != nil {
		panic(err)
	}
	_ = read

	answer, err := fb.NewReadAnswer(1, values.String("IDDQD"))
	if err != nil {
		panic(err)
	}
	_ = answer
}
```

By hand:

```go
write := frame.Frame{
	RequestID: 2,
	Body: bodies.Write{
		Key:   fields.Key("code"),
		Value: values.String("IDDQD"),
	},
}
if err := write.IsValid(); err != nil {
	panic(err)
}

read := frame.Frame{RequestID: 1, Body: bodies.Read("code")}
answer := frame.Frame{RequestID: 1, Body: bodies.ReadAnswer{Value: values.String("IDDQD")}}
```

Fields on the wire for Read — magic, version, command, RequestID, compression, Body Len, key, CRC-32C:

```go
body := bodies.Read("code")

var out buffer.Slice
out.Append(frame.MagicBytes)
fields.Version(1).Encode(&out)
body.Command().Encode(&out)
fields.RequestID(1).Encode(&out)
fields.None.Encode(&out)
out.AppendUint32(uint32(body.Size()))
body.Encode(&out)
fields.NewChecksum(out.Bytes()).Encode(&out)
```

Other bodies:

```go
fb.NewDelete(3, fields.Key("another-key"))
fb.NewPing(4)
fb.NewWriteAnswer(2)
fb.NewDeleteAnswer(3)
fb.NewPingAnswer(4)
```

## 4. Decode a frame from a stream

```go
package main

import (
	"bufio"
	"io"

	v0decoder "github.com/dejitarudemon/ignicula-wire/v0/decoder"
	v1decoder "github.com/dejitarudemon/ignicula-wire/v1/decoder"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func readFrame(r *bufio.Reader) error {
	v0 := v0decoder.NewDecoder()
	version, err := v0.DecodePreamble(r)
	if err != nil {
		return err
	}

	if version == 0 {
		_, err = v0.DecodeFrame(r)
		return err
	}

	v1 := v1decoder.NewDecoder(fields.BodyLimit(1<<20), nil)
	_, err = v1.DecodeFrame(r)
	return err
}

func serve(r io.Reader) error {
	return readFrame(bufio.NewReader(r))
}
```

## 5. Compress the body

Compression applies to the Body only. The checksum is computed over the compressed bytes. Handshake and Ping must not be compressed.

```go
package main

import (
	"bufio"
	"bytes"

	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor/compressors"
	"github.com/dejitarudemon/ignicula-wire/v1/decoder"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func main() {
	zstd, err := compressors.NewZstd(1 << 20)
	if err != nil {
		panic(err)
	}

	fb := builder.NewFrameBuilder(1 << 20)
	f, err := fb.NewWrite(2, fields.Key("blob"), values.Bytes(make([]byte, 256)))
	if err != nil {
		panic(err)
	}

	var out buffer.Slice
	if err := f.Encode(&out, zstd); err != nil {
		panic(err)
	}

	d := decoder.NewDecoder(1<<20, []compressor.Compressor{zstd})
	_, err = d.DecodeFrame(bufio.NewReader(bytes.NewReader(out.Bytes())))
	if err != nil {
		panic(err)
	}
}
```

## 6. Batch

A batch is not a transaction. Only Read, Write, and Delete are allowed. The builder assigns operation numbers starting at zero.

```go
package main

import (
	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func main() {
	fb := builder.NewFrameBuilder(1 << 20)
	batch := builder.NewBatchRequestsBuilder().
		SequentialExecution(true).
		InterruptAfterError(false).
		OneAnswer(true).
		AddRead(fields.Key("key")).
		AddWrite(fields.Key("another-key"), values.String("data"))

	f, err := fb.NewBatch(1, *batch)
	if err != nil {
		panic(err)
	}
	_ = f
}
```

By hand:

```go
f := frame.Frame{
	RequestID: 1,
	Body: bodies.Batch{
		IsSequentialExecution: true,
		InterruptAfterError:   false,
		IsOneAnswer:           true,
		Requests: []bodies.Request{
			{Number: 0, Body: bodies.Read("key")},
			{Number: 1, Body: bodies.Write{
				Key:   fields.Key("another-key"),
				Value: values.String("data"),
			}},
		},
	},
}
if err := f.IsValid(); err != nil {
	panic(err)
}
```

Flags:

| Flag | 1 | 0 |
| --- | --- | --- |
| Sequential | by number | in parallel |
| Interrupt after error | remaining requests get Request Interrupted | remaining requests continue |
| One answer | a single combined reply | replies as they complete |

The builder only checks `IsValid` and the size limit. After that it is the same `frame.Frame.Encode`. Hello, Handshake, Read/Write, and Batch show manual assembly in their sections: a frame struct or fields on the wire plus the checksum.

Canonical frames from the specification live in `v0/internal/specs` and `v1/internal/specs`.
