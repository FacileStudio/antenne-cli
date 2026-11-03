package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/FacileStudio/porte/devicelogin"
	"github.com/FacileStudio/porte/loopback"
	"golang.org/x/term"

	"github.com/FacileStudio/antenne-cli/internal/client"
	"github.com/FacileStudio/antenne-cli/internal/ui"
)

// browserAvailable reports whether opening a URL can plausibly reach a person:
// an interactive terminal, no --no-browser, an opener on PATH and a display.
func browserAvailable() bool {
	if flagNoBrowser || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	opener := "xdg-open"
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "windows":
		opener = "cmd"
	}
	if _, err := exec.LookPath(opener); err != nil {
		return false
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return os.Getenv("SSH_CONNECTION") == ""
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// ssoLogin signs in through single sign-on and returns the session token. The
// device grant is the default; an instance that has not shipped its exchange,
// or --listener, gets the loopback flow, and --no-listener the pasted code.
func ssoLogin(ctx context.Context, api *client.Client, serverURL string) (string, error) {
	if flagNoListener {
		return ssoLoginByPaste(ctx, api, serverURL)
	}
	if !flagListener {
		s, err := devicelogin.Login(ctx, devicelogin.Options{
			AppOrigin:   serverURL,
			Mount:       apiMount,
			OpenBrowser: func(link string) bool { return browserAvailable() && loopback.OpenBrowser(link) },
		})
		switch {
		case errors.Is(err, devicelogin.ErrExchangeNotMounted), errors.Is(err, devicelogin.ErrNoDeviceGrant):
			ui.Warn("this server has no device login yet, using the browser listener")
		case errors.Is(err, devicelogin.ErrAccountNotLinked):
			return "", fmt.Errorf("sign in once at %s in a browser, then run 'antenne login' again", serverURL)
		case err != nil:
			return "", err
		default:
			return s.Token, nil
		}
	}
	return ssoLoginByListener(ctx, api, serverURL)
}

// ssoLoginByListener is the loopback flow: the browser lands on a local port
// with a one-time code. A machine with no browser to land there pastes it.
func ssoLoginByListener(ctx context.Context, api *client.Client, serverURL string) (string, error) {
	if !browserAvailable() && !flagListener {
		return ssoLoginByPaste(ctx, api, serverURL)
	}

	listener, err := loopback.Listen()
	if err != nil {
		return "", fmt.Errorf("cannot open a loopback port to receive the login: %w", err)
	}
	defer listener.Close()
	listener.AppName = ssoAppName

	state, err := loopback.RandomState()
	if err != nil {
		return "", err
	}

	authURL := listener.LoginURL(serverURL, apiMount, state)
	if !loopback.OpenBrowser(authURL) {
		ui.Step("open %s", authURL)
	}
	ui.Step("waiting for the browser to sign you in to %s", serverURL)

	code, err := listener.WaitForCode(ctx, state)
	if err != nil {
		if errors.Is(err, loopback.ErrTimeout) {
			return "", fmt.Errorf("timed out waiting for the browser, run 'antenne login' again")
		}
		return "", err
	}
	return api.Exchange(ctx, code)
}

// ssoLoginByPaste leaves the loopback port off the login URL, which makes the
// server show the code on the page instead of redirecting it to 127.0.0.1.
func ssoLoginByPaste(ctx context.Context, api *client.Client, serverURL string) (string, error) {
	authURL := strings.TrimRight(serverURL, "/") + apiMount + "/auth/oidc?flow=cli"
	if !flagNoBrowser && browserAvailable() {
		loopback.OpenBrowser(authURL)
	}
	ui.Step("open %s", authURL)

	fmt.Fprint(os.Stderr, "Paste the code from that page: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("failed to read the login code: %w", err)
	}
	code := strings.TrimSpace(line)
	if code == "" {
		return "", fmt.Errorf("no login code was pasted, the code is good for sixty seconds so run 'antenne login --no-listener' again")
	}
	return api.Exchange(ctx, code)
}
