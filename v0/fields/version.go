package fields

import (
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

// Version is a protocol version number written in the Hello body or frame preamble.
type Version uint8

const (
	// VersionFieldSize is the encoded size in bytes of a [Version].
	VersionFieldSize = 1
	// CurrentVersion is the protocol version of a Hello frame.
	CurrentVersion = 0
)

// Encode writes the wire encoding of the version into buf.
func (v Version) Encode(buf buffer.Appender) {
	buf.AppendUint8(uint8(v))
}

// Size returns the encoded size in bytes.
func (v Version) Size() int {
	return VersionFieldSize
}
