Below is a strong AI agent instruction document you can give to Cursor/Copilot/Claude Code/etc. for go-tpkt.
## Project Mission: ISO Transport Service on TCP (RFC 1006)

### Purpose
This project implements the ISO Transport Service (TP0) on top of TCP, as specified in [RFC 1006](https://datatracker.ietf.org/doc/html/rfc1006). It enables ISO/CCITT session, presentation, and application protocols to operate over TCP/IP networks, facilitating interoperability and transition between protocol suites.

### Background
RFC 1006 defines a standard for offering ISO transport services using TCP/IP, reserving TCP port 102 for this purpose. By layering ISO TP0 on TCP, higher-level ISO protocols can be used in the Internet without disrupting existing TCP/IP facilities.

### Motivation
The project supports gatewaying at the transport layer, leveraging the maturity and stability of TCP/IP. This approach allows development and experience with ISO protocols while providing a graceful migration path for future transitions to ISO-based networks.

### Scope
- Implements ISO TP0 protocol on top of TCP (not on ISO/CCITT network protocol)
- Adheres to RFC 1006, supporting all aspects of ISO8072 except Quality of Service parameters
- Not a full migration or intercept document; focused on transport service emulation

### References
- RFC 1006: ISO Transport Service on top of the TCP ([link](https://datatracker.ietf.org/doc/html/rfc1006))
- ISO8072: Transport Service Definition
- ISO8073: Transport Protocol Specification

---
For details, see `spec/rfc1006.txt`.
⸻

AI Agent Instructions — go-tpkt

Mission

Implement a small, production-grade Go library for TPKT (RFC 1006).

This library must be:
	•	cleanly scoped
	•	idiomatic Go
	•	well tested
	•	safe for streaming use
	•	dependency-light
	•	reusable as a foundation for higher-level OT/industrial protocol stacks such as:
	•	COTP
	•	S7comm
	•	MMS / IEC 61850 transports over RFC 1006

The library must only own TPKT framing/parsing.
It must not implement COTP, S7, MMS, TSAP semantics, session logic, or application protocol behavior.

⸻

Repository intent

Current repo:

go-tpkt
└── spec
    └── rfc1006.txt

Target outcome:
	•	a small Go module
	•	clear public API
	•	excellent unit tests
	•	strong validation
	•	zero ambiguity around framing behavior
	•	full CI-ready quality

⸻

Protocol scope

Implement TPKT only, as specified by RFC 1006.

TPKT is a simple header + payload framing layer over TCP.

TPKT frame layout

TPKT header is always 4 bytes:
	•	byte 0: version = 3
	•	byte 1: reserved = 0
	•	bytes 2-3: total packet length in bytes, big-endian
	•	includes the 4-byte TPKT header
	•	minimum valid total length = 4

Then follows the payload:
	•	payload length = total_length - 4

In scope

Implement:
	•	frame encoding
	•	frame decoding
	•	stream reading
	•	stream writing helpers
	•	validation
	•	error classification
	•	safe handling of malformed or truncated input
	•	size limit protection

Explicitly out of scope

Do not implement:
	•	COTP parsing
	•	CR/CC/DT TPDU decoding
	•	TSAP parsing
	•	ISO session/presentation
	•	S7comm
	•	MMS
	•	protocol-specific connection logic
	•	retransmission/reassembly beyond a single TPKT frame
	•	TCP listener/server logic

⸻

Design goals

1. Keep the public API minimal

Public API should be small, obvious, and stable.

2. Be streaming-friendly

The library must work well on io.Reader / io.Writer.

3. Be strict by default

Reject malformed frames clearly and early.

4. Avoid hidden allocations where possible

Provide efficient helpers, but prioritize correctness and readable code.

5. Provide excellent errors

Errors must be:
	•	specific
	•	wrapped properly
	•	comparable when useful
	•	suitable for callers building robust protocol stacks

6. Be safe against untrusted input

Never trust length fields blindly.
Protect callers from absurd frame sizes.

⸻

Package layout

Use a single public package:

tpkt

Internal helpers are allowed if useful, but avoid overengineering.

Recommended layout:

.
├── go.mod
├── README.md
├── LICENSE
├── spec/
│   └── rfc1006.txt
├── tpkt.go
├── reader.go
├── writer.go
├── errors.go
├── doc.go
├── tpkt_test.go
├── reader_test.go
├── writer_test.go
└── fuzz_test.go

You may merge files if smaller is cleaner.

⸻

Public API requirements

Implement a clean API similar to this shape.

Constants

const (
    Version       byte = 3
    HeaderLength       = 4
)

Errors

Provide exported sentinel errors where appropriate:

var (
    ErrTooShort
    ErrInvalidVersion
    ErrInvalidReserved
    ErrInvalidLength
    ErrLengthMismatch
    ErrFrameTooLarge
)

Sentinel errors should support errors.Is.

Frame type

Use a small struct:

type Frame struct {
    Payload []byte
}

Keep it simple.
Do not add protocol-specific fields.

Optional methods:
	•	Len() int
	•	MarshalBinary() ([]byte, error)

Top-level encode/decode helpers

Implement:

func Encode(payload []byte) ([]byte, error)
func Decode(pkt []byte) ([]byte, error)
func Parse(pkt []byte) (Frame, error)

Behavior:
	•	Encode builds a full TPKT packet from payload
	•	Decode validates and returns payload only
	•	Parse validates and returns a Frame

Reader

Implement a streaming reader:

type Reader struct {
    r            io.Reader
    maxFrameSize int
}

func NewReader(r io.Reader, opts ...ReaderOption) *Reader
func (r *Reader) ReadFrame() ([]byte, error)
func (r *Reader) Read() (Frame, error)

Reader must:
	•	first read exactly 4 header bytes
	•	validate header
	•	allocate/read remaining payload
	•	handle short reads safely
	•	return io.EOF / io.ErrUnexpectedEOF appropriately
	•	enforce configurable max frame size

Writer

Implement a writer helper:

type Writer struct {
    w io.Writer
}

func NewWriter(w io.Writer) *Writer
func (w *Writer) WriteFrame(payload []byte) (int, error)
func (w *Writer) Write(f Frame) (int, error)

Behavior:
	•	write a full framed TPKT packet
	•	return bytes written for the full packet, not payload only
	•	wrap errors properly

Reader options

Implement an option pattern only if it stays small and useful:

type ReaderOption func(*Reader)

func WithMaxFrameSize(n int) ReaderOption

Provide a sensible default max size.

⸻

Validation rules

All decoders/parsers must validate:

Header checks
	•	version must be 3
	•	reserved must be 0
	•	total length must be >= 4

Buffer consistency

For non-stream parsing:
	•	declared length must equal actual slice length

Max frame size
	•	reject frames above configured max frame size
	•	protect against memory abuse

Empty payload

Allow it if total length is exactly 4, unless RFC reading in your implementation phase proves this should be rejected. If unsure, document behavior explicitly and test it.

⸻

Error handling requirements

General rules
	•	never panic on malformed input
	•	always return wrapped/contextual errors
	•	use fmt.Errorf("...: %w", err) when adding context
	•	use sentinel errors for main categories

Desired style

Good:

return nil, fmt.Errorf("decode tpkt: %w", ErrInvalidVersion)

Better when preserving detail:

return nil, fmt.Errorf("decode tpkt: version=%d: %w", b[0], ErrInvalidVersion)

Avoid
	•	vague messages like "invalid packet"
	•	losing root cause
	•	string-only error comparisons
	•	panics for user input

⸻

Interface and type guidance

Keep interfaces consumer-side

Do not invent interfaces like:

type FrameReader interface { ... }
type FrameWriter interface { ... }

unless there is a real use case.

Prefer concrete types:
	•	Reader
	•	Writer

Accept standard library interfaces:
	•	io.Reader
	•	io.Writer

Prefer value types where appropriate

Frame can be returned by value.

Avoid premature abstraction

This repo should stay lean.

⸻

Test coverage requirements

The library must have full meaningful test coverage.

That means:
	•	not just line coverage
	•	branch coverage for all validation paths
	•	edge cases
	•	streaming cases
	•	partial read behavior
	•	malformed frames
	•	boundary lengths

Required test categories

1. Encode tests
	•	empty payload
	•	small payload
	•	arbitrary payload
	•	correct version/reserved/length bytes
	•	max-size boundary behavior

2. Decode/parse tests
	•	valid frame
	•	too short input
	•	invalid version
	•	invalid reserved
	•	length < 4
	•	declared length > actual length
	•	declared length < actual length
	•	zero payload
	•	non-empty payload
	•	oversized frame

3. Reader tests
	•	valid frame from bytes.Buffer
	•	multiple concatenated frames
	•	truncated header
	•	truncated payload
	•	short/chunked reader behavior
	•	oversized frame rejected before large allocation
	•	EOF semantics

4. Writer tests
	•	correct bytes written
	•	correct framing
	•	underlying writer error propagation
	•	partial writer behavior if applicable

5. Round-trip tests
	•	payload -> encode -> decode
	•	frame -> marshal -> parse

6. Fuzz tests
Use Go fuzzing for:
	•	Decode
	•	Parse

Goal:
	•	no panics
	•	deterministic failures
	•	malformed input safely rejected

Coverage target

Aim for ~100% coverage on core package files, excluding purely impossible branches.
If something is not practically coverable, explain why in code comments or PR notes.

⸻

Testing style requirements
	•	use table-driven tests
	•	keep test names explicit
	•	avoid giant unreadable test functions
	•	test one behavior per subtest when possible
	•	compare errors with errors.Is
	•	verify exact payload/bytes where relevant

Example style:

func TestDecode(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        want    []byte
        wantErr error
    }{
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Decode(tt.input)
            if !errors.Is(err, tt.wantErr) {
                t.Fatalf("expected error %v, got %v", tt.wantErr, err)
            }
            if diff := cmp.Diff(tt.want, got); diff != "" {
                t.Fatalf("payload mismatch (-want +got):\n%s", diff)
            }
        })
    }
}

Prefer standard library only unless a tiny helper library adds strong value.
Default preference: no external deps for tests, except maybe google/go-cmp if really justified.

⸻

Documentation requirements

Package docs

Add doc.go with package-level documentation:
	•	what TPKT is
	•	what this package does
	•	what it intentionally does not do
	•	mention RFC 1006
	•	mention that upper-layer protocols are out of scope

README

Add a concise README that includes:
	•	purpose
	•	scope
	•	install
	•	example usage
	•	stream reader/writer example
	•	relation to higher-level stacks like COTP/S7/MMS

Go doc quality

All exported identifiers must have proper doc comments.

⸻

Example usage requirements

README and/or package docs should include examples like:

payload := []byte{0x02, 0xf0, 0x80}
pkt, err := tpkt.Encode(payload)

and streaming:

r := tpkt.NewReader(conn)
payload, err := r.ReadFrame()

and writing:

w := tpkt.NewWriter(conn)
_, err := w.WriteFrame(payload)


⸻

Performance expectations

This package does not need heroic optimization, but it should be sane.

Must do
	•	avoid unnecessary copies if easy
	•	avoid huge allocations on malicious length fields
	•	validate declared size before payload allocation

Nice to have
	•	benchmarks for encode/decode small frames
	•	benchmark for streaming reader

Only add benchmarks if they stay small and useful.

⸻

Style and implementation rules

Idiomatic Go

Follow standard Go style:
	•	simple naming
	•	small focused files
	•	no Java/C# style class design
	•	no unnecessary getters/setters

No overengineering

Avoid:
	•	builders
	•	inheritance-like patterns
	•	generic abstractions without need
	•	internal frameworks

Keep naming crisp

Good:
	•	Encode
	•	Decode
	•	Parse
	•	ReadFrame
	•	WriteFrame

Avoid:
	•	CreateTPKTPacketFromPayload
	•	ProcessIncomingFrameData

⸻

Backward compatibility expectations

This is an early library.
Prefer clean API correctness now over speculative compatibility.

But:
	•	keep the public API small so it stays stable later
	•	avoid exposing fields you may regret

⸻

Quality gate

Implementation is not complete until all of the following are true:
	•	go test ./... passes
	•	fuzz tests compile and run
	•	coverage is effectively full on core logic
	•	exported API has comments
	•	README explains scope clearly
	•	malformed input handling is thoroughly tested
	•	no linter-worthy error handling shortcuts
	•	no hidden protocol leakage beyond TPKT

⸻

Suggested implementation order

Phase 1 — core codec
	•	constants
	•	errors
	•	Frame
	•	Encode
	•	Decode
	•	Parse
	•	core tests

Phase 2 — stream helpers
	•	Reader
	•	Writer
	•	reader/writer tests
	•	truncation behavior tests

Phase 3 — docs and polish
	•	README
	•	package docs
	•	examples
	•	fuzz tests
	•	optional benchmarks

⸻

Non-goals

Do not:
	•	implement COTP here
	•	create a monorepo transport stack
	•	add protocol negotiation
	•	add Siemens-specific helpers
	•	add IEC 61850 helpers
	•	add PCAP parsing
	•	add sniff/replay logic

This repo must remain a clean transport framing primitive.

⸻

Definition of done

The library is done when:
	1.	it can reliably encode/decode TPKT frames,
	2.	it safely reads frames from a stream,
	3.	it handles malformed input cleanly,
	4.	it has excellent tests,
	5.	it is pleasant to import as a dependency for future go-cotp, go-s7comm, and go-mms libraries.

⸻

Final instruction to the agent

Implement the smallest clean design that fully satisfies the above.
When in doubt:
	•	choose simpler APIs
	•	choose stricter validation
	•	choose better tests
	•	choose clearer errors
	•	avoid future regret in exported surface

⸻

I can also turn this into a ready-to-save AGENTS.md or CLAUDE.md file next.