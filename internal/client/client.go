// Package client talks to an Antenne instance's HTTP API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SessionCookie is the cookie Antenne's session travels in. The API authenticates
// by cookie only, so the CLI stores the value and sends it back on every call.
const SessionCookie = "nook_session"

// Error carries the API's own error code alongside its message, so a caller can
// branch on the code rather than parse prose.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// Unauthenticated reports whether the instance rejected the session.
func (e *Error) Unauthenticated() bool { return e.Status == http.StatusUnauthorized }

// Client is a connection to one instance.
type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// New builds a client. The timeout is generous because a delivery test waits on
// a third party — Matrix, SMTP — that Antenne cannot hurry.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) request(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, payload)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "antenne-cli")
	if c.Token != "" {
		request.AddCookie(&http.Cookie{Name: SessionCookie, Value: c.Token})
	}
	return request, nil
}

// do performs a request and decodes the response into out, if given.
//
// The body is read as text and parsed defensively rather than through a
// streaming decoder: Antenne serves its dashboard from the same origin, so a
// mistyped path returns 200 and HTML, and a bare JSON syntax error would hide
// that the URL was simply wrong.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	request, err := c.request(ctx, method, path, body)
	if err != nil {
		return err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("cannot reach %s — %w", c.BaseURL, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return decodeError(raw, response.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s answered with something that is not JSON — check the URL points at an Antenne instance", c.BaseURL)
	}
	return nil
}

func decodeError(raw []byte, status int) error {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Error.Message != "" {
		return &Error{Status: status, Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	return &Error{Status: status, Code: "unknown", Message: fmt.Sprintf("HTTP %d", status)}
}

// Login exchanges the admin password for a session token.
func (c *Client) Login(ctx context.Context, password string) (string, error) {
	request, err := c.request(ctx, http.MethodPost, "/api/login", map[string]string{"password": password})
	if err != nil {
		return "", err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("cannot reach %s — %w", c.BaseURL, err)
	}
	defer response.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", decodeError(raw, response.StatusCode)
	}

	for _, cookie := range response.Cookies() {
		if cookie.Name == SessionCookie && cookie.Value != "" {
			return cookie.Value, nil
		}
	}

	// An instance with no admin password configured answers 200 and sets no
	// cookie, because every caller is already served as the admin.
	return "", nil
}

// Logout revokes the stored session server-side.
func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/logout", struct{}{}, nil)
}

// Session reports who the instance thinks the caller is.
func (c *Client) Session(ctx context.Context) (Session, error) {
	var session Session
	err := c.do(ctx, http.MethodGet, "/api/session", nil, &session)
	return session, err
}

// PublicSession reports whether the instance requires a password at all,
// without holding one.
func (c *Client) PublicSession(ctx context.Context) (Session, error) {
	var session Session
	err := c.do(ctx, http.MethodGet, "/api/session/public", nil, &session)
	return session, err
}

// Health probes the instance. It needs no session, so it is the right call for
// telling "unreachable" apart from "not logged in".
func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/api/health", nil, nil)
}

// Settings returns the configuration, the runtime status and whether a usable
// Matrix target exists. Secrets come back redacted.
func (c *Client) Settings(ctx context.Context) (SettingsResponse, error) {
	var payload SettingsResponse
	err := c.do(ctx, http.MethodGet, "/api/settings", nil, &payload)
	return payload, err
}

// Events returns a filtered page of the activity log.
func (c *Client) Events(ctx context.Context, query EventQuery) (EventPage, error) {
	values := url.Values{}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Offset > 0 {
		values.Set("offset", strconv.Itoa(query.Offset))
	}
	if query.Search != "" {
		values.Set("q", query.Search)
	}
	if query.Source != "" {
		values.Set("source", query.Source)
	}

	path := "/api/events"
	switch {
	case query.ProviderID != "":
		path = "/api/events/provider/" + url.PathEscape(query.ProviderID)
	case query.TargetID != "":
		path = "/api/events/target/" + url.PathEscape(query.TargetID)
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var page EventPage
	err := c.do(ctx, http.MethodGet, path, nil, &page)
	return page, err
}

// TestAlert sends a test through the whole pipeline, routing rules included, so
// it stays silent unless a target selected the system provider.
func (c *Client) TestAlert(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/test-alert", struct{}{}, nil)
}

// TestTarget sends straight to one target, bypassing the routing rules.
func (c *Client) TestTarget(ctx context.Context, targetID string) error {
	return c.do(ctx, http.MethodPost, "/api/test-delivery-target", map[string]string{"targetId": targetID}, nil)
}

// TestProvider verifies a provider's connection. Only IMAP has one to test.
func (c *Client) TestProvider(ctx context.Context, providerID string) error {
	return c.do(ctx, http.MethodPost, "/api/test-provider-connection", map[string]string{"providerId": providerID}, nil)
}

// Replay re-sends a logged event to one target.
func (c *Client) Replay(ctx context.Context, eventID, targetID string) error {
	path := "/api/events/" + url.PathEscape(eventID) + "/replay"
	return c.do(ctx, http.MethodPost, path, map[string]string{"targetId": targetID}, nil)
}

// PoolStats returns the live state of the the Antenne bus.
func (c *Client) PoolStats(ctx context.Context) (PoolStats, error) {
	var stats PoolStats
	err := c.do(ctx, http.MethodGet, "/api/pool/stats", nil, &stats)
	return stats, err
}

// Stream follows the live event feed until the context is cancelled, calling
// onEvent for each entry.
//
// The transport is server-sent events, so this is a long-lived response body
// read frame by frame rather than a WebSocket. A frame that will not decode is
// skipped instead of ending the stream: one malformed entry should not stop a
// tail that has been running for an hour.
func (c *Client) Stream(ctx context.Context, onEvent func(Event)) error {
	request, err := c.request(ctx, http.MethodGet, "/api/events/stream", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/event-stream")

	// No client timeout on the stream: the whole point is that it stays open.
	streaming := &http.Client{Transport: c.http.Transport}
	response, err := streaming.Do(request)
	if err != nil {
		return fmt.Errorf("cannot reach %s — %w", c.BaseURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return decodeError(raw, response.StatusCode)
	}

	return readSSE(response.Body, func(payload []byte) {
		var event Event
		if json.Unmarshal(payload, &event) == nil {
			onEvent(event)
		}
	})
}
