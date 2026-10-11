package fields

import (
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

// Version is a protocol version number written in the frame preamble.
type Version uint8

const (
	// VersionFieldSize is the encoded size in bytes of a [Version].
	VersionFieldSize = 1
	// CurrentVersion is the protocol version used when encoding new frames.
	CurrentVersion = 1
)

// Encode writes the wire encoding of the version into buf.
func (r Version) Encode(buf buffer.Appender) {
	buf.AppendUint8(uint8(r))
}

// Size returns the encoded size in bytes.
func (r Version) Size() int {
	return VersionFieldSize
}
