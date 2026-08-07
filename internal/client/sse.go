package client

import (
	"bufio"
	"bytes"
	"io"
)

// maxFrame caps one server-sent event. An entry carries a full envelope, which
// can hold a third-party payload, so the limit is generous — but it exists, so
// a misbehaving server cannot grow the buffer without bound.
const maxFrame = 8 << 20

// readSSE parses a server-sent event stream, calling onData once per event with
// its concatenated data payload.
//
// It implements only the part of the SSE grammar Antenne emits — `data:` lines
// terminated by a blank line — and skips comments and every other field. A
// full parser would carry event ids and retry hints that this feed never sends.
func readSSE(body io.Reader, onData func([]byte)) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxFrame)

	var payload []byte

	for scanner.Scan() {
		line := scanner.Bytes()

		// A blank line terminates the event.
		if len(bytes.TrimSpace(line)) == 0 {
			if len(payload) > 0 {
				onData(payload)
				payload = nil
			}
			continue
		}

		// A leading colon is a comment, which servers send as a keep-alive.
		if line[0] == ':' {
			continue
		}

		field, value, found := bytes.Cut(line, []byte(":"))
		if !found || string(field) != "data" {
			continue
		}

		// One optional leading space after the colon belongs to the framing,
		// not to the value.
		value = bytes.TrimPrefix(value, []byte(" "))

		if len(payload) > 0 {
			payload = append(payload, '\n')
		}
		payload = append(payload, value...)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	// A stream that ends mid-event still has one to deliver.
	if len(payload) > 0 {
		onData(payload)
	}
	return nil
}
