# Changelog

## 1.2.0 - 2026-10-11

Added:

- Handshake Answer lists granted protocol versions the client may use without re-authentication

Changed:

- `fields.Version` is `uint8` (was `uint32`) in both v0 and v1

## 1.1.0 - 2026-10-09

Changed:

- A frame with an unsupported protocol version closes the connection instead of returning an error
- Error code 1 is Command Not Implemented: a valid protocol command that this node has not implemented (was Unsupported Version)

## 1.0.1 - 2026-10-02

Added:

- Extend on bodies.BatchRequestsBuilder, which appends another batch's requests before the frame is built
- Merge on bodies.BatchResultsBuilder, which lets a BatchAnswer take in other BatchAnswers before the frame is built

## 1.0.0 — 2026-09-26

We fucking did it.

First release of the Go codec for Ignicula.

- Hello (protocol version 0) and working version 1
- Typed values on the wire
- Version chosen per frame after Hello
