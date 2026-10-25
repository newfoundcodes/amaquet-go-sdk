package protocol

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	Magic               = "AMQT"
	Version        byte = 1
	HeaderSize          = 24
	MaxPayloadSize      = 64 << 20
)

// Kind: Provides the Kind operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Kind byte

const (
	KindRequest  Kind = 1
	KindResponse Kind = 2
	KindEvent    Kind = 3
)

// Opcode: Provides the Opcode operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Opcode uint16

const (
	OpCommand     Opcode = 1
	OpHello       Opcode = 2
	OpAuth        Opcode = 3
	OpSubscribe   Opcode = 4
	OpUnsubscribe Opcode = 5
	OpCancel      Opcode = 6
)

// Frame: Provides the Frame operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Frame struct {
	Kind      Kind
	Flags     uint16
	Opcode    Opcode
	RequestID uint64
	Payload   []byte
}

// WriteFrame: Write Frame using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func WriteFrame(w io.Writer, f Frame) error {
	if len(f.Payload) > MaxPayloadSize {
		return fmt.Errorf("payload exceeds %d bytes", MaxPayloadSize)
	}

	h := make([]byte, HeaderSize)
	copy(h[0:4], []byte(Magic))

	h[4] = Version
	h[5] = byte(f.Kind)

	binary.BigEndian.PutUint16(h[6:8], f.Flags)
	binary.BigEndian.PutUint16(h[8:10], uint16(f.Opcode))
	binary.BigEndian.PutUint64(h[12:20], f.RequestID)
	binary.BigEndian.PutUint32(h[20:24], uint32(len(f.Payload)))

	if _, e := w.Write(h); e != nil {
		return e
	}

	if len(f.Payload) > 0 {
		_, e := w.Write(f.Payload)
		return e
	}

	return nil
}

// ReadFrame: Read Frame using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func ReadFrame(r io.Reader) (Frame, error) {
	f, release, err := ReadFrameLimitBudget(r, MaxPayloadSize, nil)
	if release != nil {
		release()
	}

	return f, err
}

// ReadFrameLimit: Read Frame Limit using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func ReadFrameLimit(r io.Reader, maxPayload int64) (Frame, error) {
	f, release, err := ReadFrameLimitBudget(r, maxPayload, nil)
	if release != nil {
		release()
	}

	return f, err
}

// ReadFrameLimitBudget: Read Frame Limit Budget using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func ReadFrameLimitBudget(r io.Reader, maxPayload int64, acquire func(int64) (func(), error)) (Frame, func(), error) {
	if maxPayload <= 0 || maxPayload > MaxPayloadSize {
		maxPayload = MaxPayloadSize
	}

	h := make([]byte, HeaderSize)
	if _, e := io.ReadFull(r, h); e != nil {
		return Frame{}, nil, e
	}

	if string(h[0:4]) != Magic {
		return Frame{}, nil, fmt.Errorf("invalid magic")
	}

	if h[4] != Version {
		return Frame{}, nil, fmt.Errorf("unsupported protocol version %d", h[4])
	}

	n := int64(binary.BigEndian.Uint32(h[20:24]))
	if n > maxPayload {
		return Frame{}, nil, fmt.Errorf("payload too large: %d > %d", n, maxPayload)
	}

	var release func()
	if acquire != nil && n > 0 {
		var err error
		release, err = acquire(n)

		if err != nil {
			return Frame{}, nil, err
		}
	}

	p := make([]byte, int(n))
	if _, e := io.ReadFull(r, p); e != nil {
		if release != nil {
			release()
		}

		return Frame{}, nil, e
	}

	return Frame{
		Kind:      Kind(h[5]),
		Flags:     binary.BigEndian.Uint16(h[6:8]),
		Opcode:    Opcode(binary.BigEndian.Uint16(h[8:10])),
		RequestID: binary.BigEndian.Uint64(h[12:20]),
		Payload:   p,
	}, release, nil
}

// ReadBufferedFrame: Read Buffered Frame using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func ReadBufferedFrame(r *bufio.Reader) (Frame, error) {
	return ReadFrame(r)
}

// Request: Submit an operation with a request ID and await its matching response or a timeout or connection error.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Request struct {
	Command string                     `json:"command"`
	Args    map[string]json.RawMessage `json:"args,omitempty"`
}

// Response: Provides the Response operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Response struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Error: Represents an Amaquet error with diagnostic information for callers; avoid logging credentials or private payloads.
// Represents or validates the 24-byte Amaquet native protocol envelope and its length-limited payload.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Encode: Encode using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeRequest: Decode Request using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Protocol v1 frames use a fixed 24-byte header and a maximum 64 MiB payload; reject malformed frames.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func DecodeRequest(b []byte) (Request, error) {
	var r Request
	e := json.Unmarshal(b, &r)

	return r, e
}
