package bodies

import (
	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/err"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

const (
	// LoginLenFieldSize is the encoded size in bytes of the login length prefix.
	LoginLenFieldSize = 4
	// HashFieldSize is the encoded size in bytes of the handshake hash.
	HashFieldSize = 32
	// CompressionLenFieldSize is the encoded size in bytes of the compression count.
	CompressionLenFieldSize = 1
	// MaxCompressionsPerOneHandshake is the maximum number of compression codes in one handshake.
	MaxCompressionsPerOneHandshake = 255
)

var _ body.Body = Handshake{}

// Handshake is a [fields.Handshake] body carrying login, hash, and offered compressions.
type Handshake struct {
	// Login is the client identity.
	Login string
	// Hash is the authentication hash.
	Hash [32]byte
	// Compressions lists compression algorithms the client can use.
	Compressions []fields.Compression
}

// NewHandshake returns a Handshake after dropping [fields.None] and duplicate compression codes.
func NewHandshake(login string, hash [32]byte, compressions []fields.Compression) Handshake {
	return Handshake{
		Login:        login,
		Hash:         hash,
		Compressions: filter(compressions),
	}
}

// filter drops zeros and duplicate values.
func filter[T ~uint8](values []T) []T {
	filtered := make([]T, 0, len(values))
	used := make(map[T]struct{}, len(values))

	for _, value := range values {
		if _, ok := used[value]; !ok && value != 0 {
			used[value] = struct{}{}
			filtered = append(filtered, value)
		}
	}

	return filtered
}

// Size returns the encoded body size in bytes.
func (h Handshake) Size() int {
	size := min(len(h.Compressions), MaxCompressionsPerOneHandshake)

	return LoginLenFieldSize + cap(h.Hash) + len(h.Login) + CompressionLenFieldSize + size*fields.CompressionFieldSize
}

// Encode writes the wire encoding of the body into buf.
func (h Handshake) Encode(buf buffer.Appender) {
	buf.AppendUint32(uint32(len(h.Login)))
	buf.AppendString(h.Login)
	buf.Append(h.Hash[:])
	buf.AppendUint8(uint8(min(len(h.Compressions), MaxCompressionsPerOneHandshake)))

	for i, fields := range h.Compressions {
		if i >= MaxCompressionsPerOneHandshake {
			break
		}

		fields.Encode(buf)
	}
}

// Command returns [fields.Handshake].
func (h Handshake) Command() fields.Command {
	return fields.Handshake
}

// IsValid reports whether the body satisfies protocol rules.
// Compression codes are not checked because custom algorithms are allowed.
// The compression list must not exceed [MaxCompressionsPerOneHandshake].
func (h Handshake) IsValid() error {
	if len(h.Compressions) > MaxCompressionsPerOneHandshake {
		return err.NewValidationError(
			"too many compressions",
			"compressions", len(h.Compressions),
			"target", h.Command(),
		)
	}

	return nil
}
