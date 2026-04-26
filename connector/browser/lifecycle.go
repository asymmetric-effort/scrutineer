package browser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// browserProcess manages a running browser instance.
type browserProcess struct {
	cmd    *exec.Cmd
	wsURL  string
	cancel context.CancelFunc
}

// chromiumArgs returns command-line arguments for launching Chromium in headless mode.
func chromiumArgs(headless bool, extraArgs []string) []string {
	args := []string{
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-breakpad",
		"--disable-component-extensions-with-background-pages",
		"--disable-component-update",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-hang-monitor",
		"--disable-ipc-flooding-protection",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-renderer-backgrounding",
		"--disable-sync",
		"--disable-translate",
		"--metrics-recording-only",
		"--no-startup-window",
		"--password-store=basic",
		"--use-mock-keychain",
		"--remote-debugging-port=0",
	}

	if headless {
		args = append(args, "--headless=new")
	}

	args = append(args, extraArgs...)
	return args
}

// firefoxArgs returns command-line arguments for launching Firefox.
func firefoxArgs(headless bool, extraArgs []string) []string {
	args := []string{
		"--no-remote",
		"--new-instance",
		"-wait-for-browser",
		"--remote-debugging-port", "0",
	}

	if headless {
		args = append(args, "--headless")
	}

	args = append(args, extraArgs...)
	return args
}

// webkitArgs returns command-line arguments for launching WebKit.
func webkitArgs(headless bool, extraArgs []string) []string {
	args := []string{
		"--inspector-pipe",
	}

	if headless {
		args = append(args, "--headless")
	}

	args = append(args, extraArgs...)
	return args
}

// buildArgs returns the browser launch arguments for the given browser type.
func buildArgs(browserType string, headless bool, extraArgs []string) []string {
	switch browserType {
	case "firefox":
		return firefoxArgs(headless, extraArgs)
	case "webkit":
		return webkitArgs(headless, extraArgs)
	default:
		return chromiumArgs(headless, extraArgs)
	}
}

// launchBrowser starts a browser process and extracts the WebSocket debugger URL.
func launchBrowser(ctx context.Context, browserPath, browserType string, headless bool, extraArgs []string) (*browserProcess, error) {
	args := buildArgs(browserType, headless, extraArgs)

	launchCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(launchCtx, browserPath, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("browser: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("browser: start %s: %w", browserPath, err)
	}

	// Read stderr to find the WebSocket URL.
	wsURL, err := extractWSURL(stderr, 10*time.Second)
	if err != nil {
		cancel()
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("browser: extract ws url: %w", err)
	}

	return &browserProcess{
		cmd:    cmd,
		wsURL:  wsURL,
		cancel: cancel,
	}, nil
}

// extractWSURL reads from the browser's stderr looking for the WebSocket debugger URL.
func extractWSURL(r interface{ Read([]byte) (int, error) }, timeout time.Duration) (string, error) {
	scanner := bufio.NewScanner(r)
	ch := make(chan string, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if idx := strings.Index(line, "ws://"); idx >= 0 {
				// Extract URL from the line.
				url := line[idx:]
				// Trim trailing whitespace or non-URL characters.
				if end := strings.IndexAny(url, " \t\r\n"); end >= 0 {
					url = url[:end]
				}
				ch <- url
				return
			}
		}
		ch <- ""
	}()

	select {
	case url := <-ch:
		if url == "" {
			return "", errors.New("browser: no websocket url found in output")
		}
		return url, nil
	case <-time.After(timeout):
		return "", errors.New("browser: timeout waiting for websocket url")
	}
}

// kill terminates the browser process.
func (bp *browserProcess) kill() error {
	if bp.cancel != nil {
		bp.cancel()
	}
	if bp.cmd != nil && bp.cmd.Process != nil {
		_ = bp.cmd.Process.Kill()
		return bp.cmd.Wait()
	}
	return nil
}
