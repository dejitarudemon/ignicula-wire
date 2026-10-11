package bodies

import (
	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

var _ body.Answer = HandshakeAnswer{}

const (
	// VersionLenFieldSize is the encoded size in bytes of the granted-versions count.
	VersionLenFieldSize = 1
	// MaxVersionsPerOneHandshakeAnswer is the maximum number of granted versions in one handshake answer.
	MaxVersionsPerOneHandshakeAnswer = 255
)

// HandshakeAnswer is a successful reply to [fields.Handshake].
// It lists accepted compressions and protocol versions the client may use without re-authenticating.
type HandshakeAnswer struct {
	// Compressions lists compression algorithms the server accepts.
	Compressions []fields.Compression
	// Versions lists protocol versions granted for use without re-authentication.
	Versions []fields.Version
}

// NewHandshakeAnswer returns a HandshakeAnswer after dropping zeros and duplicates
// from compressions and versions.
func NewHandshakeAnswer(compressions []fields.Compression, versions []fields.Version) HandshakeAnswer {
	return HandshakeAnswer{
		Compressions: filter(compressions),
		Versions:     filter(versions),
	}
}

// Size returns the encoded body size in bytes.
func (h HandshakeAnswer) Size() int {
	compressions := min(len(h.Compressions), MaxCompressionsPerOneHandshake)
	versions := min(len(h.Versions), MaxVersionsPerOneHandshakeAnswer)

	size := ResultFieldSize + CompressionLenFieldSize + VersionLenFieldSize + h.IsResponseTo().Size()

	return size + compressions*fields.CompressionFieldSize + versions*fields.VersionFieldSize
}

// Encode writes the wire encoding of the body into buf.
func (h HandshakeAnswer) Encode(buf buffer.Appender) {
	buf.Append(ResultOK)
	h.IsResponseTo().Encode(buf)
	buf.AppendUint8(uint8(min(len(h.Compressions), MaxCompressionsPerOneHandshake)))

	for i, compression := range h.Compressions {
		if i >= MaxCompressionsPerOneHandshake {
			break
		}

		compression.Encode(buf)
	}

	buf.AppendUint8(uint8(min(len(h.Versions), MaxVersionsPerOneHandshakeAnswer)))

	for i, version := range h.Versions {
		if i >= MaxVersionsPerOneHandshakeAnswer {
			break
		}

		version.Encode(buf)
	}
}

// Command returns [fields.Answer].
func (h HandshakeAnswer) Command() fields.Command {
	return fields.Answer
}

// IsResponseTo returns [fields.Handshake].
func (h HandshakeAnswer) IsResponseTo() fields.Command {
	return fields.Handshake
}

// IsValid reports whether the body satisfies protocol rules.
// Compression codes are not checked because custom algorithms are allowed.
// The compression list must not exceed [MaxCompressionsPerOneHandshake].
// The versions list must not exceed [MaxVersionsPerOneHandshakeAnswer].
func (h HandshakeAnswer) IsValid() error {
	if len(h.Compressions) > MaxCompressionsPerOneHandshake {
		return err.NewValidationError(
			"too many compressions",
			"compressions", len(h.Compressions),
			"target", h.Command(),
		)
	}

	if len(h.Versions) > MaxVersionsPerOneHandshakeAnswer {
		return err.NewValidationError(
			"too many versions",
			"versions", len(h.Versions),
			"target", h.Command(),
		)
	}

	return nil
}
