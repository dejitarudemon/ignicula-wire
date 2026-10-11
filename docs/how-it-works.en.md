Language: [Русский](how-it-works.md) · [English](how-it-works.en.md)

# How it works

Ignicula is a binary protocol on a byte stream (usually TCP, optionally TLS). This library encodes and decodes frames. The socket, storage, and access policy stay in the application.

The full rules are in the [specification](specs.en.md). API walkthroughs are in the [tutorials](tutorial.en.md).

## Connection lifetime

```text
client                         server
   |                              |
   |-------- Hello v0 ----------->|
   |<------- Hello v0 ------------|
   |   version intersection       |
   |-------- Handshake v1 ------->|
   |<------- Answer --------------|
   |                              |
   |   Read / Write / Delete /    |
   |   Ping / Batch  (RequestID)  |
```

1. The client opens TCP (and TLS if the server requires it) **before** Hello.
2. Both nodes exchange version-0 Hello frames and take the intersection of the lists. Version 0 is not a working version.
3. The client sends Handshake (`RequestID = 0`) with a login, an Argon2id hash, and offered compression algorithms.
4. The server replies with a Handshake Answer (accepted compressions and versions usable without re-authentication) or a protocol error.
5. Working commands follow. One frame has one `RequestID`. Several requests may be in flight at once.

If there are no common versions, the client closes the connection. After a drop, every unfinished request is considered unsuccessful.

## Frame

Shared layout for every version:

```text
| Magic 0A DB (2) | Version (1) | Headers | Body | Checksum (4) |
```

Byte order is big-endian.

**Version 0 (Hello).** Headers are a single `Version Len` byte. The body is that many one-byte version numbers. The checksum is CRC-32/XFER over the whole frame except itself. The frame is exactly `8 + Version Len` bytes.

```text
| 0A DB | 00 | N | v1 .. vN | XFER |
```

**Version 1.** Headers are the command, `RequestID`, compression code, and body length.

```text
| 0A DB | 01 | Command | RequestID | Compression | Body Len | Body | CRC-32C |
```

The checksum is Castagnoli (CRC-32C) over the whole frame except itself. If the body is compressed, the sum is checked **before** decompression. The receiver reads exactly `Body Len + 4` bytes after the headers.

`DecodePreamble` peeks at the first three bytes and does not advance the cursor. The `Version` byte selects the decoder: `v0` or `v1`. The version can change on every frame: Hello is v0, then working commands use v1 or another agreed version on the same connection.

## Asynchrony

`RequestID` multiplexes requests on one connection. The value `0` is reserved for Handshake and its answer. Reusing a live `RequestID` yields Request Conflict: the first request continues, the second is rejected.

The answer always carries the same `RequestID` as the request. The client matches replies by that field, not by frame order.

## Types and commands

Version 1 tags values with a type code: bytes, typed/untyped array, int, uint, float64 (IEEE 754), UTF-8 string, JSON.

Commands:

| Code | Name | Who sends it |
| --- | --- | --- |
| 0 | Handshake | client only, once after Hello |
| 1 | Answer | either node, only as a reply |
| 2 | Read | client |
| 3 | Write | client |
| 4 | Delete | client |
| 5 | Batch | client |
| 6 | Ping | either node |

Read and Delete place the key in the whole body. Write prefixes the key with a length and a type tag.

## Compression

Code `0` means no compression. The specification defines zstd (`1`) and s2 (`2`); other codes are reserved. Only the Body is compressed. Handshake and Ping must not be compressed. The decoder takes a compressor map and rejects an unknown code with Unsupported Compression.

Receive order: headers → body + checksum → CRC check → decompress → parse the command.

## Batch

A batch is not a transaction. Only Read, Write, and Delete are allowed inside. Three flags:

- sequential or parallel;
- whether to stop remaining operations after an error;
- one combined Answer or replies as they complete.

Each nested request has a `Request Number` inside the batch. The answer repeats those numbers. If replies arrive in several frames, the client sums `Results Len` until it matches the original `Requests Len`.

## Errors

A protocol error is an Answer with a code, a 16-byte `TracebackID`, and optional Details. The traceback ties the client reply to a server log line.

Library-local errors (bad magic, EOF, a v0 checksum mismatch, a builder size limit) are not encoded on the wire. The application decides whether to close the connection or reply with a protocol error.

## Limits and heartbeat

The specification sets field ceilings (Body up to 4 GiB, 255 Hello versions, 255 compression codes). A server may tighten them. In this library the body limit is passed to `NewDecoder` / `NewFrameBuilder`.

The Ping interval is not fixed. The specification suggests: the server sends Ping every 30 s of idle time and drops the connection after 100 s without a reply; an idle client sends Ping every 15 s.
