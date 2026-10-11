Language: [Русский](specs.md) · [English](specs.en.md)

# Terminology

`Node` — an independent participant in network communication under this protocol. A node may be a client or a server.

`Request` — a logically distinct message according to the frame structure and the format of a specific version, which a node executes as part of a logical operation.

A request may be:
- `initiating` — the first in a request–response pair.
- `responding` — the second in a request–response pair.

`Frame` — a generalized structure containing the data sufficient for correct transmission of a request.

# Basic Frame Structure

```text
| Magic bytes (2 bytes) | Version (1 byte) | Headers | Body | Checksum (4 bytes) |
```

Where:

- `Magic bytes` — a unique sequence indicating that this frame belongs to the Ignicula protocol (0A DB).
- `Version` — the protocol version (from 0 to 255).
- `Headers` — service data and flags defined by the protocol version (hereinafter also headers).
- `Body` — the main message data.
- `Checksum` — the frame checksum. Concrete implementations themselves determine what exactly the checksum covers.

By default, **all** fields and messages use big-endian byte order.

# Establishing a Common Version

Regardless of the implementation, every connection starts with a version-0 service frame (hereinafter the Hello request) of the following format:

```text
| 0A DB | 00 | Version Len (1 byte) | Version 1 | Version 2 | ...  | Version N | CheckSum |
```

Where:

- `Version Len` — the number of protocol versions supported by the node.
- `Version N` — a specific supported version.

In the basic frame structure, the Headers field is represented by a single Version Len field. Versions are represented as the Body.

The checksum algorithm is `CRC-32/XFER` (poly=`0x000000AF`, init=`0`, refin=`false`, refout=`false`, xorout=`0`, check=`0xBD0BE338`). Checksum covers the entire frame except itself.

The frame occupies exactly `8 + Version Len` bytes: 2 magic bytes, 1 version byte, 1 Version Len byte, Version Len version bytes, and 4 checksum bytes.

**Author's note**: duplicate versions are allowed but highly discouraged. A repeated version number is ignored. Version 0 is not a working version and is not counted in the list of supported versions.

## Connection Establishment Algorithm

1. The client establishes a TCP connection with the server. TLS or other encryption of network traffic is optional and depends on the specific server implementation.
2. The client sends a Hello request listing the supported versions.
3. The server replies with a message of the same format containing its supported versions.
4. The client remembers the versions supported by both the client and the server and may communicate using only those.
5. The client and the server then proceed according to the protocol version.

If the client does not support any version that the server supports, the connection cannot be established; the client must close the connection, otherwise the server will close it after some finite time.

Example Hello requests from the client to the server and back:

```text
Client -> Server

Hex:

0A DB 00 03 01 02 03 6E 38 99 00

Decode:
	Version: 0 (Hello)
	Versions: 1, 2, 3


Server -> Client

Hex:

0A DB 00 04 01 04 07 0B 3B 78 5D 98

Decode:
	Version: 0 (Hello)
	Versions: 1, 4, 7, 11

Common versions: 1
```

A node reads exactly `8 + Version Len` bytes of a single Hello frame. If fewer versions arrive than declared, the connection is closed. Bytes after the frame belong to the next message.

# Versions

**Author's note**: version 0 is not a standalone working version and is used only for Hello requests. If the client sends a frame of a version the server does not support, the server must close the connection.

## 1

### Data Types

The data type codes and their encoding formats are listed below. Nodes must guarantee that data conforms to the requirements of the stated data types.

#### Byte Sequence

Code: 0

Example: 0A 12 33 45

Notes: -

Conformance check:
1. The actual length matches the declared length.

Encoding format:

```text
| Value Len (4 bytes) | Value |
```

Where:

- `Value Len` — the length of the byte sequence.
- `Value` — the byte sequence.

Encoding algorithm:
1. Write the length of the byte sequence.
2. Write the byte sequence.

Decoding algorithm:
1. Read the first 4 bytes — Value Len.
2. Read the next Value Len bytes.

Encoding/decoding example:

```text
Hex: 00 00 00 04 0A 12 33 45

Decode:
	Value Len: 4
	Value: 0A 12 33 45
```

#### Typed Array

Code: 1

Example: {1, 2, 3}

Notes: each Value must be expanded for the types "[Byte Sequence](#byte-sequence)", "[Typed Array](#typed-array)", "[Untyped Array](#untyped-array)", "[String](#string)", "[JSON](#json)" as the structure described for each of those types.

Conformance check:
1. The declared array length matches the actual length.
2. Each element conforms to the declared type.

Encoding format:

```text
| Array Len (4 bytes) | Value Type (1 byte) | Value 1 | Value 2 | ... | Value N |
```

Where:

- `Array Len` — the number of elements.
- `Value Type` — the array element type.
- `Value N` — the array elements.

Encoding algorithm:
1. Write the number of array elements.
2. Write the element type once.
3. Write all elements sequentially, encoding them according to their type.

Decoding algorithm:
1. Read the first 4 bytes — Array Len.
2. Read 1 byte — Value Type.
3. Decode the elements sequentially according to their decoding algorithm, checking each element for conformance to its type.

Encoding/decoding example:

```text
Hex: 00 00 00 02 03 00 00 00 00 00 00 00 01 00 00 00 00 00 00 00 02

Decode:
	Array Len: 2
	Value Type: int
	Value 1: 1
	Value 2: 2
```

#### Untyped Array

Code: 2

Example: {1, "hello world", 12.3}

Notes: each Value must be expanded for the types "[Byte Sequence](#byte-sequence)", "[Typed Array](#typed-array)", "[Untyped Array](#untyped-array)", "[String](#string)", "[JSON](#json)" as the structure described for each of those types.

Conformance check:
1. The declared array length matches the actual length.
2. Each element conforms to the declared type of that element.

Encoding format:

```text
| Array Len (4 bytes) | Value 1 Type (1 byte) | Value 1 | Value 2 Type (1 byte) | Value 2 | ... | Value N |
```

Where:

- `Array Len` — the number of elements.
- `Value Type N` — the type of array element number N.
- `Value N` — array element number N.

Encoding algorithm:
1. Write the number of array elements.
2. Write the element type.
3. Write the element according to the encoding algorithm of its type.
4. Repeat steps 2–3 until the elements are exhausted.

Decoding algorithm:
1. Read the first 4 bytes — Array Len.
2. Read 1 byte — Value Type.
3. Decode and check the element according to its type.
4. Repeat steps 2–3 until the elements are exhausted.

Encoding/decoding example:

```text
Hex: 00 00 00 03 03 00 00 00 00 00 00 00 01 06 00 00 00 0B 68 65 6C 6C 6F 20 77 6F 72 6C 64 05 40 28 99 99 99 99 99 9A

Decode:
	Array Len: 3
	Value 1:
		Value Type: int
		Value: 1
	Value 2:
		Value Type: string
		Value:
			Value Len: 11
			Value: hello world
	Value 3:
		Value Type: float
		Value: 12.3
```

#### Integer

Code: 3

Example: -24

Notes: the value lies in the range from -9223372036854775808 to 9223372036854775807.

Conformance check: none.

Encoding format (Big Endian):

```text
| Value (8 bytes) |
```

Where:

- `Value` — the numeric value. The sign is represented in two's complement.

Encoding algorithm:
1. Write the numeric value. If the numeric value does not require all 8 bytes for storage, pad the remaining bytes with zeros.

Decoding algorithm:
1. Read the first 8 bytes.

Encoding/decoding example:

```text
Hex: 00 00 00 00 00 00 00 10

Decode:
	Value: 16
```

#### Unsigned Integer

Code: 4

Example: 42

Notes: the value lies in the range from 0 to 18446744073709551615.

Conformance check: none.

Encoding format:

```text
| Value (8 bytes) |
```

Where:

- `Value` — the numeric value.

Encoding algorithm:
1. Write the numeric value. If the numeric value does not require all 8 bytes for storage, pad the remaining bytes with zeros.

Decoding algorithm:
1. Read the first 8 bytes.

Encoding/decoding example:

```text
Hex: 00 00 00 00 00 00 00 20

Decode:
	Value: 32
```

#### Floating-Point Number

Code: 5

Example: 31.42

Notes: a positive value lies in the range from 2.2250738585072014e-308 to 1.7976931348623157e+308. A negative value lies in the range from -1.7976931348623157e+308 to -2.2250738585072014e-308. The number is encoded in IEEE 754 format.

Conformance check: none.

Encoding format:

```text
| Value (8 bytes) |
```

Where:

- `Value` — the numeric value in IEEE 754 format.

Encoding algorithm:
1. Convert the number to a byte sequence according to the IEEE 754 format.
2. Write the byte sequence.

Decoding algorithm:
1. Read the first 8 bytes.
2. Convert the read byte sequence according to the IEEE 754 format.

Encoding/decoding example:

```text
Hex: 40 28 DC 28 F5 C2 8F 5C

Decode:
	Value: 12.43
```

#### String

Code: 6

Example: Hello world

Notes: it is a Unicode (UTF-8) string.

Conformance check:
1. The actual length matches the declared length.

Encoding format:

```text
| Value Len (4 bytes) | Value |
```

Where:

- `Value Len` — the string length in bytes.
- `Value` — the string.

Encoding algorithm:
1. Convert the string to a byte sequence.
2. Write the length of the byte sequence.
3. Write the byte sequence.

Decoding algorithm:
1. Read the first 4 bytes — Value Len.
2. Read the next Value Len bytes.
3. Convert the byte sequence according to the encoding.

Encoding/decoding example:

```text
Hex: 00 00 00 05 69 64 64 71 64

Decode:
	Value Len: 5
	Value: iddqd
```

#### JSON

Code: 7

Example:
```json
{
	"this": "is example"
}
```

Notes: it is a Unicode (UTF-8) string.

Conformance check:
1. Basic JSON syntax.

Encoding format:

```text
| Value Len (4 bytes) | Value |
```

Where:

- `Value Len` — the string length in bytes.
- `Value` — the JSON document.

Encoding algorithm:
1. Convert the document to a string.
2. Convert the string to a byte sequence.
3. Write the length of the byte sequence.
4. Write the byte sequence.

Decoding algorithm:
1. Read the first 4 bytes — Value Len.
2. Read the next Value Len bytes.
3. Convert the byte sequence according to the encoding into a string.
4. Check that the string conforms to basic JSON document syntax.

Encoding/decoding example:

```text
Hex: 00 00 00 15 7B 22 74 68 69 73 22 3A 22 69 73 20 65 78 61 6D 70 6C 65 22

Decode:
	Value Len: 21
	Value:
		Key: this
		Value: is example
```

### Errors

The error codes and the Details fields they return are listed below.

Note: other codes are reserved: they may be used in a private implementation of this protocol, or they will be added later in this version.

#### No Hello

Error code: 0

Examples:
1. The client did not send a Hello request after connecting.

Cause: version mismatch, missing Hello request.

Notes: this error is not sent to the client; it is recorded only in the logs. The connection is closed.

How to fix: after establishing the TCP connection, send a Hello request.

Details structure: -

#### Command Not Implemented

Code: 1

Examples:
1. The client sends a Delete request, but the server has not implemented handling of that command.

Cause: the server has not implemented the logic to process a request for the specified command.

Notes: -

How to fix: do not use this method to work with the server.

Details structure: -

#### Unexpected Command

Code: 2

Examples:
1. The client sends Read in response to a Ping from the server.

Cause: unexpected command. The receiving node expected a different command.

Notes: -

How to fix: send the command that the receiving node expects.

Details structure:

```text
| Expected Command (1 byte) |
```

Where:

- `Expected Command` — the expected command. Takes values from 0 to 255.

#### Unsupported Command

Code: 3

Examples:
1. The client sent a frame with command 8.

Cause: unsupported command. The specified command is not in the command list of the current version.

Notes: -

How to fix: use commands from the current protocol version, or switch to another version where that command is supported (if such a version exists and is supported by both the client and the server).

Details structure: -

#### Requests Conflict

Code: 4

Examples:
1. The client sends a request with Request ID 1 and immediately another with Request ID 1.

Cause: a request was received with a Request ID that is already registered at the processing node.

Notes: the first request (in order of receipt time) is not interrupted. The second request is not processed (an error is returned). The response to the first request must arrive in the normal order.

How to fix: retry the request with a different Request ID.

Details structure: -

#### Unsupported Compression

Code: 5

Examples:
1. The client or the server uses a compression algorithm not supported by the other node.

Cause: unsupported compression algorithm. The specified algorithm is not supported by the receiving side.

Notes: —

How to fix: send a request using an algorithm supported by the receiving node.

Details structure: -

#### Body Limit Is Exceeded

Code: 6

Examples:
1. The client sent a request with Body Len = 10 while the server's message size limit is 8 bytes.

Cause: the Body size exceeds the server's configured body size limit.

Notes: desynchronization is possible.

How to fix: reconnect if the connection is dropped, reduce the data size, or use compression.

Details structure:

```text
| Current Body Size Limit (4 bytes) |
```

Where:

- `Current Body Size Limit` — the current body size limit. Takes values from 0 to 4 294 967 295.

#### Mismatched CheckSum

Code: 7

Examples:
1. The client sent a request with corrupted Headers or Body.
2. The client sent a request with Body Len = 8, although the actual length is 9 bytes (while the server's Body size limit is 12 bytes).

Cause: the frame checksum does not match the declared value.

Notes: desynchronization is possible if the error is in Body Len.

How to fix: reconnect if the connection is dropped, send a request with a correct Checksum.

Details structure: -

#### Internal Error

Code: 8

Examples:
1. The server went out of array bounds and this error was not handled.

Cause: an unhandled error inside the server.

Notes: it can be anything.

How to fix: see the logs.

Details structure: -

#### Malformed Value

Code: 9

Examples:
1. The JSON document structure is invalid.

Cause: the value contents do not match the format of their type.

Notes: -

How to fix: send a request with a correct data type specification.

Details structure:

```text
| Info Message |
```

Where:
- `Info Message` — a textual error description in UTF-8 format. The message may be absent. It occupies the remainder of Details.

#### Not Found

Code: 10

Examples:
1. The client sent a Read request with a key that the server does not have.

Cause: no value was found for the Key.

Notes: this code indicates that there is no value for the key. It does not interrupt operations in a batch.

How to fix: no correction is required. Further actions are determined by the client's business logic.

Details structure: -

#### Prohibited Compression

Code: 11

Examples:
1. The client sent a Ping request with compression.

Cause: compression is used in requests where it is prohibited.

Notes: —

How to fix: send the request without compression.

Details structure: -

#### Request Interrupted

Code: 12

Examples:
1. The server was executing a request but is being forcibly shut down.
2. During execution of a Batch request with the "Interrupt on error" flag set to 1, one of the operations failed due to an error. In that case the error is reported in the corresponding Result Body.

Cause: the server registered the request but cannot execute it because execution was interrupted.

Notes: -

How to fix: check that the server is available. In the case of a batch, check the error that caused subsequent operations to be interrupted.

Details structure: -

#### Batch Limit Is Exceeded

Code: 13

Examples:
1. The client sent a Batch request containing 10 operations while the server's limit is 8 operations in a single batch.

Cause: the Requests Len value exceeds the server's limit on the number of operations in a single batch.

Notes: -

How to fix: split the batch into several separate requests.

Details structure:

```text
| Current Batch Size Limit (4 bytes) |
```

Where:

- `Current Batch Size Limit` — the current limit on the number of operations in a Batch request. Takes values from 0 to 4 294 967 295.

#### Unexpected Command In Batch

Code: 14

Examples:
1. The client sent a Batch request in which the request numbered 2 specified the Handshake command.

Cause: the Command value of one of the Requests is not Read, Write, or Delete.

Notes: -

How to fix: remove the request from the batch. Send it separately if needed.

Details structure:

```text
| Request Number (4 bytes) |
```

Where:

- `Request Number` — the number of the request with an invalid Command.

#### Invalid Request ID

Code: 15

Examples:
1. The client sent a Handshake request with Request ID = 1.
2. The client sent a Read request with Request ID = 0.

Cause: a RequestID is used that is not intended for this request.

Notes: -

How to fix: change the Request ID for this request.

Details structure: -

#### Unauthorized

Code: 16

Examples:
1. The client sent a Handshake request with an incorrect login and/or password.

Cause: incorrect login and/or password in the Handshake request.

Notes: -

How to fix: enter the correct login and password.

Details structure: -

#### Restricted Request

Code: 17

Examples:
1. The client sent a Read request for data it does not have access to.

Cause: the current client does not have access to the specified command and Body.

Notes: -

How to fix: contact the system administrator or another person responsible for access-control policies.

Details structure: -


### Compression

The list of compression algorithms and their corresponding codes is given below.

| Code | Algorithm |
| --- | -------- |
| 0   | -        |
| 1   | zstd     |
| 2   | s2       |
Other codes are reserved: they may be used in a private implementation of this protocol, or they will be added later in this version.

### Headers

```text
| Command (1 byte) | RequestID (4 bytes) | Compression (1 byte) | Body Len (4 bytes) |
```

Where:

- `Command` — the command number (see [Command](#command)).
- `RequestID` — a unique identifier of a client request that the node is not currently processing.
- `Compression` — the compression used for the Body.
- `Body Len` — the Body length in bytes.

#### Command

A byte denoting the command number.

Existing commands:

0. `Handshake` — the first request, initiated only by the client after the Hello request, in which it transmits the information required for further operation: supported compression methods, etc. See [Handshake Algorithm](#handshake-algorithm) for details.
1. `Answer` — a response to a request. It cannot be used as a standalone request: responding to a nonexistent request must return an [Unexpected Command](#unexpected-command) error. It may be used by both the client and the server.
2. `Read` — a request to retrieve data. May be used only by the client. See [Reading Data](#data-read-algorithm) for details.
3. `Write` — a request to write data. May be used only by the client. See [Writing Data](#data-write-algorithm) for details.
4. `Delete` — a request to delete data. May be used only by the client. See [Deleting Data](#data-delete-algorithm) for details.
5. `Batch` — a set of requests. It is not a transaction. May be used only by the client. See [Batches and Their Use](#batch-request-submission-algorithm) for details.
6. `Ping` — a request to check availability. May be used by both the client and the server. See [Availability Check](#network-availability-check-algorithm) for details.

Other commands in the current version are reserved and may be used by specific implementations of this protocol version. Otherwise, requests with reserved commands must be answered with an [Unsupported Command](#unsupported-command) error.

#### Request ID

An identifier unique to the current connection, from 0 to 4 294 967 295. Used for asynchronous protocol operation.

The value 0 is used only for Handshake and for Answer to Handshake.

#### Compression

A value from 0 to 255. Used to denote the algorithm used to compress the Body. 0 denotes no compression. See [Compression](#compression) for the list of supported algorithms.

#### Body Len

A value from 0 to 4 294 967 295 (4 GiB). Used to denote the Body size (after compression, if compression is used). The receiving side reads exactly Body Len + 4 bytes (the checksum).

### Body

Contains the request data and depends on Command. If compression is used, after verifying the checksum the inverse transform according to the compression algorithm must be performed.

### CheckSum

Verifies frame integrity. Covers the entire frame excluding itself. The algorithm used is Castagnoli (CRC-32C). If the Body is compressed, the checksum is verified before the inverse transform according to the compression algorithm.

### Handshake Algorithm

A client that has decided to establish a connection with the server must perform the handshake procedure.

#### Opening a Connection

See [Connection Establishment Algorithm](#connection-establishment-algorithm).

#### Sending a Handshake Request

The client sends a Handshake request containing the following headers (Headers):

- `Command` — 0
- `RequestID` — 0
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| Login Len (4 bytes) | Login | Hash (32 bytes) | Supported Compressions Len (1 byte) | Compression 1 | Compression 2 | ... | Compression N |
```

Where:

- `Login Len` — the length of the login used to connect.
- `Login` — the login used to connect. It is a UTF-8 string.
- `Hash` — the password hash used to connect. The hash function used is Argon2id.
- `Supported Compressions Len` — the number of supported compression methods (see [Compression](#compression)). Takes values from 0 to 255.
- `Compression N` — the compression algorithms supported by the client.

**Author's note**: the server may support anonymous connections. In that case Login Len becomes 0, and Hash is filled with zero bytes.

Example Handshake request with supported algorithms Zstd and S2:

```text
Hex:

0A DB 01 00 00 00 00 00 00 00 00 00 2B 00 00 00 04 75 73 65 72 01 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 02 01 02 7A 39 15 BF

Decode:
	Version: 1
	Command: Handshake
	Request ID: 0
	Compression: -
	Login: user
	Hash: 0x01 0x02 0x00 ... 0x00
	Supported compression algorithms: Zstd, S2
	CheckSum: 7A 39 15 BF
```

#### Server Response

Upon receiving the request, the server validates and parses it.

Two scenarios are possible:

1. Valid request

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — 0
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 01 | 00 | Supported Compressions Len | Compression 1 | Compression 2 | ... | Compression N | Granted Versions Len | Version 1 | Version 2 | ... | Version N |
```

Where:

- `Supported Compression Len` — the number of supported compression methods from the list provided by the client. Takes values from 0 to 255.
- `Compression N` — the compression algorithms supported by the server.
- `Granted Versions Len` — the number of versions the server allows without re-authentication. Takes values from 0 to 255.
- `Version N` — a version granted for use without re-authentication.

Example Answer request with Zstd support and access to version 1:

```text
Hex:
0A DB 01 01 00 00 00 00 00 00 00 00 06 01 00 01 01 01 01 52 6E 3E 35

Decode:
	Version: 1
	Command: Answer
	Request ID: 0
	Compression: -
	Result: OK
	Responded To: Handshake
	Supported compression algorithms: Zstd
	Granted versions: 1
	CheckSum: 52 6E 3E 35
```

2. Request with problems

In this case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — 0
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

**Author's note**: here and in the examples below, Traceback ID is interpreted as a binary UUID.

Example Answer request with an [Unexpected Command](#unexpected-command) error, caused by the client not sending a Handshake request after the Hello request:

```text
Hex:

0A DB 01 01 00 00 00 00 00 00 00 00 14 00 00 02 6B AE 36 80 76 95 47 42 92 AF 0A 6F EC 33 98 25 00 F0 8E 28 51

Decode:
	Version: 1
	Command: Answer
	Request ID: 0
	Compression: -
	Result: NOT OK
	Error:
		Unexpected Command
		Traceback ID: 6bae3680-7695-4742-92af-0a6fec339825
		Details:
    		Expected Command: Handshake
	CheckSum: F0 8E 28 51
```

### Data Read Algorithm

#### Sending a Read Request

The client sends a Read request containing the following headers (Headers):

- `Command` — 2
- `Request ID` — unique: there must be no requests currently in progress with the same value
- `Compression` — optional, an algorithm supported by the server
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| Key |
```

Where:

- `Key` — a binary sequence corresponding to the key that will be looked up.

There is no need to specify the Key length: Key occupies the entire Body.

Example Read request without compression with Request ID = 1 and key (Key) = "some-key":

```text
Hex:

0A DB 01 02 00 00 00 01 00 00 00 00 08 73 6F 6D 65 2D 6B 65 79 E1 5A 14 00

Decode:
	Version: 1
	Command: Read
	Request ID: 1
	Compression: -
	Key: some-key
	CheckSum: E1 5A 14 00
```

#### Server Response

Upon receiving the request, the server validates and parses it.

Three scenarios are possible:

1. Valid request, the data exists.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Read request
- `Compression` — optional, an algorithm supported by the client
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 01 | 02 | Value Type (1 byte) | Value |
```

Where:

- `Value Type` — the value type. See [Data Types](#data-types).
- `Value` — the value in the format of its type.

Example Answer request without compression with Request ID = 1 and value (Value) = "some-data":

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 10 01 02 06 00 00 00 09 73 6F 6D 65 2D 64 61 74 61 5E 79 B8 83

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: OK
	Responded To: Read
	Data:
		Type: string
		Value: some-data
	CheckSum: 5E 79 B8 83
```

Example Answer request without compression with Request ID = 10 and value (Value) = {1, "hello world", 2.1}:

```text
Hex:

0A DB 01 01 00 00 00 0A 00 00 00 00 29 01 02 02 00 00 00 03 03 00 00 00 00 00 00 00 01 06 00 00 00 0B 68 65 6C 6C 6F 20 77 6F 72 6C 64 05 40 00 CC CC CC CC CC CD 34 E8 E0 75

Decode:
	Version: 1
	Command: Answer
	Request ID: 10
	Compression: -
	Result: OK
	Responded To: Read
	Data:
		Type: untyped array
		Value:
			1:
				Type: int
				Value: 1
			2:
				Type: string
				Value: hello world
			3:
				Type: float
				Value: 2.1
	CheckSum: 34 E8 E0 75
```

2. Valid request, the data does not exist.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Read request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 00 | 00 0A | TracebackID (16 bytes) |
```

which corresponds to a [Not Found](#not-found) error message with a Traceback identifier.

Example Answer request with Request ID = 1:

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 13 00 00 0A F1 38 9C 1F A2 2F 40 82 A2 79 D2 01 7C 60 F8 20 88 EE CC AA

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: NOT OK
	Error:
		Not Found
		Traceback ID: f1389c1f-a22f-4082-a279-d2017c60f820
		Details: -
	CheckSum: 88 EE CC AA
```

3. Invalid request or an internal error.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Read request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details.
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

Example Answer request with Request ID = 1 and an [Internal Error](#internal-error):

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 13 00 00 08 A5 18 B2 C1 CC 27 49 27 94 09 6A 10 15 DA 67 B1 9C 90 B4 33

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: NOT OK
	Error:
		Internal Error
		Traceback ID: a518b2c1-cc27-4927-9409-6a1015da67b1
		Details: -
	CheckSum: 9C 90 B4 33
```

### Data Write Algorithm

#### Sending a Write Request

The client sends a Write request containing the following headers (Headers):

- `Command` — 3
- `Request ID` — unique: there must be no requests currently in progress with the same value
- `Compression` — optional, an algorithm supported by the server
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| Key Len (4 bytes) | Key | Value Type (1 byte) | Value |
```

Where:

- `Key Len` — the key size in bytes.
- `Key` — a binary sequence corresponding to the key under which the value will be written.
- `Value Type` — the value type. See [Data Types](#data-types).
- `Value` — the value in the format of its type.

Example Write request without compression with Request ID = 2, key (Key) = "code" and value (Value) = "IDDQD":

```text
Hex:

0A DB 01 03 00 00 00 02 00 00 00 00 12 00 00 00 04 63 6F 64 65 06 00 00 00 05 49 44 44 51 44 38 0A 5A 23

Decode:
	Version: 1
	Command: Write
	Request ID: 2
	Compression: -
	Key: code
	Data:
		Type: string
		Value: IDDQD
	CheckSum: 38 0A 5A 23
```

Example Write request without compression with Request ID = 1, key (Key) = "vector" and value (Value) = {1, 2, 3}:

```text
Hex:

0A DB 01 03 00 00 00 01 00 00 00 00 28 00 00 00 06 76 65 63 74 6F 72 01 00 00 00 03 03 00 00 00 00 00 00 00 01 00 00 00 00 00 00 00 02 00 00 00 00 00 00 00 03 D7 99 DF 3D

Decode:
	Version: 1
	Command: Write
	Request ID: 1
	Compression: -
	Key: vector
	Data:
		Type: typed array
		Element Type: Int
		Value: 1, 2, 3
	CheckSum: D7 99 DF 3D
```

#### Server Response

Upon receiving the request, the server validates and parses it.

Two scenarios are possible:

1. Valid request.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Write request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 01 | 03 |
```

Example Answer request with Request ID = 2:

```text
Hex:

0A DB 01 01 00 00 00 02 00 00 00 00 02 01 03 A0 05 87 3E

Decode:
	Version: 1
	Command: Answer
	Request ID: 2
	Compression: -
	Result: OK
	Responded To: Write
	CheckSum: A0 05 87 3E
```

2. Invalid request or an internal error.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Write request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details.
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

Example Answer request with Request ID = 2 and an [Internal Error](#internal-error):

```text
Hex:

0A DB 01 01 00 00 00 02 00 00 00 00 13 00 00 08 77 29 53 66 99 93 4D C3 9B 99 19 24 11 EE 8C 9D 90 1B 23

Decode:
	Version: 1
	Command: Answer
	Request ID: 2
	Compression: -
	Result: NOT OK
	Error:
		Internal Error
		Traceback ID: 77295366-9993-4dc3-9b99-192411ee5b8c
		Details: -
	CheckSum: 9D 90 1B 23
```

### Data Delete Algorithm

#### Sending a Delete Request

The client sends a Delete request containing the following headers (Headers):

- `Command` — 4
- `Request ID` — unique: there must be no requests currently in progress with the same value
- `Compression` — optional, an algorithm supported by the server
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| Key |
```

Where:

- `Key` — a binary sequence corresponding to the key that will be deleted.

There is no need to specify the Key length: Key occupies the entire Body.

Example Delete request without compression with Request ID = 3 and key (Key) = "another-key":

```text
Hex:

0A DB 01 04 00 00 00 03 00 00 00 00 0B 61 6E 6F 74 68 65 72 2D 6B 65 79 F4 C7 FF 81

Decode:
	Version: 1
	Command: Delete
	Request ID: 3
	Compression: -
	Key: another-key
	CheckSum: F4 C7 FF 81
```

#### Server Response

Upon receiving the request, the server validates and parses it.

Two scenarios are possible:

1. Valid request.

In the context of deletion, whether the key actually exists on the server does not matter. The server must not raise an error ([Not Found](#not-found)).

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Delete request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 01 | 04 |
```

Example Answer request with Request ID = 3:

```text
Hex:

0A DB 01 01 00 00 00 03 00 00 00 00 02 01 04 3D F3 9E F2

Decode:
	Version: 1
	Command: Answer
	Request ID: 3
	Compression: -
	Result: OK
	Responded To: Delete
	CheckSum: 3D F3 9E F2
```

2. Invalid request or an internal error.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Delete request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details.
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

Example Answer request with Request ID = 3 and an [Internal Error](#internal-error):

```text
Hex:

0A DB 01 01 00 00 00 03 00 00 00 00 13 00 00 08 CA D1 C1 02 CA C5 4D DF 9B 61 EB 00 81 8E 25 D1 4F 23 68 80

Decode:
	Version: 1
	Command: Answer
	Request ID: 3
	Compression: -
	Result: NOT OK
	Error:
		Internal Error
		Traceback ID: cad1c102-cac5-4ddf-9b61-eb00818e25d1
		Details: -
	CheckSum: 4F 23 68 80
```

### Network Availability Check Algorithm

#### Sending a Ping Request

The client or the server sends a Ping request containing the following headers (Headers):

- `Command` — 6
- `Request ID` — unique: there must be no requests currently in progress with the same value
- `Compression` — 0
- `Body Len` — 0

The client or the server transmits nothing in the Body.

Example Ping request with Request ID = 4:

```text
Hex:

0A DB 01 06 00 00 00 04 00 00 00 00 02 A2 1B 87 9B

Decode:
	Version: 1
	Command: Ping
	Request ID: 4
	Compression: -
	CheckSum: A2 1B 87 9B
```

#### Request Response

Upon receiving the request, the server or the client validates and parses it.

Two scenarios are possible:

1. Valid request.

In that case the server/client sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Ping request
- `Compression` — 0
- `Body Len` — 0

In the Body the server or the client transmits data in the following format:

```text
| 01 | 06 |
```

Example Answer request with Request ID = 4:

```text
Hex:

0A DB 01 01 00 00 00 04 00 00 00 00 02 01 06 26 91 EB 01

Decode:
	Version: 1
	Command: Answer
	Request ID: 4
	Compression: -
	Responded To: Ping
	CheckSum: 26 91 EB 01
```

2. Invalid request or an internal error.

In that case the server/client sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Ping request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

In the Body the server transmits data in the following format:

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details.
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

Example Answer request with Request ID = 4 and an [Internal Error](#internal-error):

```text
Hex:

0A DB 01 01 00 00 00 04 00 00 00 00 13 00 00 08 3D 8E D3 BF B8 D1 47 45 A5 DB F8 50 30 90 90 2A C9 04 2A 26

Decode:
	Version: 1
	Command: Answer
	Request ID: 4
	Compression: -
	Result: NOT OK
	Error:
		Internal Error
		Traceback ID: 3d8ed3bf-b8d1-4745-a5db-f8503090902a
		Details: -
	CheckSum: C9 04 2A 26
```

### Batch Request Submission Algorithm

#### Sending a Batch Request

The client sends a Batch request containing the following headers (Headers):

- `Command` — 5
- `RequestID` — unique: there must be no requests currently in progress with the same value
- `Compression` — optional, an algorithm supported by the server
- `Body Len` — depends on the actual Body length

In the Body the client transmits data in the following format:

```text
| Flags (1 byte) | Requests Len (4 bytes) |  Request 1 | Request 2 | ... | Request N |
```

Where:

- `Flags` — execution configuration flags. See below for details.
- `Requests Len` — the number of requests in the batch.
- `Request N` — request N. See the request format below.


**Flags**

These are the batch execution settings. Flag bits:

0. `Request execution order`. 1 — sequential (in Request Number order). 0 — parallel.
1. `Interrupt on error`. 1 — interrupt further execution (a [Request Interrupted](#request-interrupted) error will be returned for the remaining unexecuted requests). 0 — return the error; the remaining operations are not interrupted.
2. `Result return`. 1 — return the results of all requests once. 0 — return results in execution order (not necessarily in Request Number order if the "request execution order" flag is set to 0).
3. Reserved.
4. Reserved.
5. Reserved.
6. Reserved.
7. Reserved.

**Request**

One of the requests included in the batch. Encoding format:

```text
| Request Number (4 bytes) | Command (1 byte) | Request Body Len (4 bytes) | Request Body |
```

Where:

- `Request Number` — a unique operation number within the current batch.
- `Command` — only a Read, Delete, or Write operation.
- `Request Body Len` — the request body size.
- `Request Body` — a Body whose format is defined for Read, Delete, and Write requests.

Example Batch request with two operations:
- Read request with Request Number = 1 and Key = "key"
- Write request with Request Number = 2, Key = "another-key" and Value = "data"

The following flags are used:
- return the response once.
- skip the operation on error.
- execute operations sequentially.

Headers:
- `RequestID` = 1
- `Compression` = 0

```text
Hex:

0A DB 01 05 00 00 00 01 00 00 00 00 32 05 00 00 00 02 00 00 00 01 02 00 00 00 03 6B 65 79 00 00 00 02 03 00 00 00 18 00 00 00 0B 61 6E 6F 74 68 65 72 2D 6B 65 79 06 00 00 00 04 64 61 74 61 DE 84 44 1C

Decode:
	Version: 1
	Command: Batch
	Request ID: 1
	Compression: -
	Flags:
		One answer
		Continue with errors
		Execute sequentially
	Requests:
		1:
			Command: Read
			Key: key
		2:
			Command: Write
			Key: another-key
			Data:
				Type: string
				Value: data
	CheckSum: DE 84 44 1C
```

#### Server Response

Upon receiving the request, the server validates and parses it.

Two scenarios are possible:

1. Valid request. The batch is accepted for processing.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Batch request
- `Compression` — optional, an algorithm supported by the client
- `Body Len` — depends on the actual Body length

```text
| 01 | 05 | Results Len (4 bytes) | Result 1 | Result 2 | ... | Result N |
```

Where:

- `Results Len` — the number of responses in the frame.
- `Result N` — the result of executing request N. See the format below.

**Result**

One of the request results from the batch. Encoding format:

```text
| Request Number (4 bytes) | Result Body Len (4 bytes) | Result Body |
```

Where:

 - `Request Number` — a unique operation number in the batch.
 - `Result Body Len` — the Result Body length.
 - `Result Body` — a Body defined for successful and unsuccessful responses to Read, Delete, and Write requests (see [Data Read Algorithm](#data-read-algorithm), [Data Write Algorithm](#data-write-algorithm), [Data Delete Algorithm](#data-delete-algorithm)).

**Author's note**: if the "Result return" flag is set to 1, there will be exactly one Answer request. Otherwise the server may "return results" one at a time using the same structure, or in several sequences. The client must itself keep track of the number of received results using the total Requests Len sent by the client and the Results Len sent by the server. The server must respond to every request in the batch.

Example Answer request without compression for a request with Request ID = 1 and the following flags set:

- return the response once.
- skip the operation on error.
- execute operations sequentially

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 26 01 05 00 00 00 02 00 00 00 01 00 00 00 0E 01 02 06 00 00 00 07 6D 65 73 73 61 67 65 00 00 00 02 00 00 00 02 01 03 B7 2D 2D F4

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: OK
	Responded To: Batch
	Batch:
		Request:
			Number: 1
			Result: OK
			Responded To: Read
			Data:
				Type: string
				Value: message
		Request:
			Number: 2
			Result: OK
			Responded To: Write
	CheckSum: B7 2D 2D F4
```

Example Answer request without compression for a request with Request ID = 1 and the following flags set:

- return the response once.
- skip the operation on error.
- execute operations sequentially

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 3C 01 05 00 00 00 02 00 00 00 01 00 00 00 13 00 00 08 38 DD 5C 39 24 DB 45 A9 85 09 FF E6 C6 5C 7F 42 00 00 00 02 00 00 00 13 00 00 0C 49 84 98 C5 CF 19 40 C9 95 38 15 5C 27 A3 CF F1 01 50 3D 7B

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: OK
	Responded To: Batch
	Batch:
		Request:
			Number: 1
			Result: NOT OK
			Error:
				Internal Error
				Traceback ID: 38dd5c39-24db-45a9-8509-ffe6c65c7f42
				Details: -
		Request:
			Number: 2
			Result: NOT OK
				Error:
					Request Interrupted
					Traceback ID: 498498c5-cf19-40c9-9538-155c27a3cff1
					Details: -
	CheckSum: 01 50 3D 7B
```

2. Invalid request. The batch is not accepted for processing.

In that case the server sends an Answer request containing the following headers (Headers):

- `Command` — 1
- `RequestID` — the same as in the Batch request
- `Compression` — 0
- `Body Len` — depends on the actual Body length

```text
| 00 | Error Code (2 bytes) | Traceback ID (16 bytes) | Details |
```

Where:

- `Error Code` — the error code. Takes values from 0 to 65 535. See [Errors](#errors) for details.
- `Traceback ID` — a unique error identifier.
- `Details` — a detailed error description that depends on the error number.

Example Answer request with Request ID = 1 and an [Internal Error](#internal-error):

```text
Hex:

0A DB 01 01 00 00 00 01 00 00 00 00 13 00 00 08 E5 3A 39 93 7B 7A 47 EC B3 82 7C 2B 05 E3 D6 DD 53 81 F9 D8

Decode:
	Version: 1
	Command: Answer
	Request ID: 1
	Compression: -
	Result: NOT OK
	Error:
		Internal Error
		Traceback ID: e53a3993-7b7a-47ec-b382-7c2b05e3d6dd
		Details: -
	CheckSum: 53 81 F9 D8
```

### Timeouts and Heartbeat

The protocol does not fix the interval for checking network availability with the [Ping](#network-availability-check-algorithm) command, nor the wait timeout.

Recommendations:
- For the server:
	- send a Ping every 30 seconds if there have been no requests from the client.
	- close the connection if the client shows no signs of activity: a response to the Ping request or any other request within 100 seconds after sending the first Ping request.
- For the client:
	- in case of idle time, send Ping requests to the server every 15 s.

### Reconnection

If the connection is dropped, the client may reconnect. All requests unfinished at the moment of the drop are considered unsuccessful: there will be no response to them. The client itself decides whether to retry the request.

### Encryption

This protocol does not fix a specific encryption algorithm and leaves the implementation to the users.

If the server uses encryption, the client must prepare a secure connection before sending the Hello request.

This protocol recommends using TLS to secure the connection.

### Limits

If no limits are specified for a frame field, the server may impose its own limits at the application level.

Current protocol limits:

- `Body Len`: 0..4 294 967 295.
- `RequestID`: 1..4 294 967 295 (the value 0 is reserved).
- `Version`: 0..255.
- `Requests Len`: 0..4 294 967 295.
- `Results Len`: 0..4 294 967 295.
- `Supported Compressions Len`: 0..255.
- `Supported Versions Len`: 0..255.
- `Error Code`: 0..65 535.
- `Current Body Size Limit`: 0..4 294 967 295.
- `Current Batch Size Limit`: 0..4 294 967 295.
