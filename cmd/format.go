package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/FacileStudio/nook-cli/internal/client"
	"github.com/FacileStudio/nook-cli/internal/ui"
)

var markupPattern = regexp.MustCompile(`<[^>]+>`)

// plainText renders the small HTML subset Nook's alert messages carry as text.
// The CLI never interprets the markup: an alert body is third-party text.
func plainText(value string) string {
	text := strings.NewReplacer("<br/>", " ", "<br>", " ", "</p>", " ").Replace(value)
	text = markupPattern.ReplaceAllString(text, " ")
	text = strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'",
	).Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

// relativeTime renders a timestamp as an age, which is what a log reader
// actually wants, falling back to the clock past a day.
func relativeTime(raw string) string {
	at, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}

	elapsed := time.Since(at)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	}
	return at.Local().Format("2 Jan 15:04")
}

// clockTime renders a timestamp as a wall clock, for the live feed where every
// line is seconds old and "just now" would say nothing.
func clockTime(raw string) string {
	at, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return at.Local().Format("15:04:05")
}

// eventTitle is the one line that describes an event.
//
// A system event's `event` field is an internal label while its message is the
// sentence, so the message reads better as the headline; every other source is
// the other way round.
func eventTitle(event client.Event) string {
	message := ""
	if event.Message != nil {
		message = plainText(*event.Message)
	}

	if event.Source == "system" && message != "" {
		return message
	}
	if event.Event != "" {
		return event.Event
	}
	if message != "" {
		return message
	}
	return event.Source
}

// deliverySummary states what happened to an event, in the fewest words that
// stay unambiguous when the row is scanned rather than read.
func deliverySummary(event client.Event) string {
	if len(event.Deliveries) == 0 {
		return ui.Dim("not routed")
	}

	delivered, failed := event.Delivered(), event.Failed()
	switch {
	case failed == 0:
		return ui.Ok(fmt.Sprintf("%d ok", delivered))
	case delivered == 0:
		return ui.Failed(fmt.Sprintf("%d failed", failed))
	}
	return fmt.Sprintf("%s %s", ui.Ok(fmt.Sprintf("%d ok", delivered)), ui.Failed(fmt.Sprintf("%d failed", failed)))
}

// truncate keeps a table column from wrapping, which would break the alignment
// every other row depends on.
func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}
