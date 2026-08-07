package client

import (
	"strings"
	"testing"
)

func collect(stream string) []string {
	var frames []string
	_ = readSSE(strings.NewReader(stream), func(payload []byte) {
		frames = append(frames, string(payload))
	})
	return frames
}

func TestOneEventPerBlankLine(t *testing.T) {
	frames := collect("data: {\"a\":1}\n\ndata: {\"a\":2}\n\n")

	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2: %q", len(frames), frames)
	}
	if frames[0] != `{"a":1}` || frames[1] != `{"a":2}` {
		t.Errorf("frames = %q", frames)
	}
}

// A keep-alive comment must not be delivered as an event, or a quiet feed would
// print a blank line every heartbeat.
func TestCommentsAreSkipped(t *testing.T) {
	frames := collect(": keep-alive\n\ndata: {\"a\":1}\n\n")

	if len(frames) != 1 || frames[0] != `{"a":1}` {
		t.Errorf("frames = %q, want just the event", frames)
	}
}

// The SSE grammar allows an event to span several data lines, joined by a
// newline. Antenne does not send those today; the parser must not corrupt one if
// it ever does.
func TestMultiLineDataIsJoined(t *testing.T) {
	frames := collect("data: {\"a\":\ndata: 1}\n\n")

	if len(frames) != 1 || frames[0] != "{\"a\":\n1}" {
		t.Errorf("frames = %q", frames)
	}
}

// A stream cut mid-event still has one to deliver — the alternative is silently
// dropping the last thing that happened before a restart.
func TestATruncatedStreamDeliversItsLastEvent(t *testing.T) {
	frames := collect("data: {\"a\":1}\n")

	if len(frames) != 1 || frames[0] != `{"a":1}` {
		t.Errorf("frames = %q", frames)
	}
}

// Exactly one space after the colon is framing, not payload. Dropping more
// would corrupt a value that legitimately begins with whitespace.
func TestOnlyOneLeadingSpaceIsStripped(t *testing.T) {
	frames := collect("data:  padded\n\n")

	if len(frames) != 1 || frames[0] != " padded" {
		t.Errorf("frames = %q, want one leading space kept", frames)
	}
}
