//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDevServerServesReloadsAndKeepsProductionOutputClean verifies the dev flow end-to-end.
func TestDevServerServesReloadsAndKeepsProductionOutputClean(t *testing.T) {
	projectRoot := t.TempDir()
	port := freeTCPPort(t)
	writeProjectFile(t, projectRoot, "veta.yaml", `
build:
  output: dist
  clean: true
dev:
  host: 127.0.0.1
  port: `+fmt.Sprint(port)+`
  watch:
    - content
`)
	writeProjectFile(t, projectRoot, "content/message.txt", "Initial")
	writeProjectFile(t, projectRoot, "templates/status.html", "Initial Include")
	writeProjectFile(
		t,
		projectRoot,
		"templates/base.html",
		`<html><body><main>{{ page.content }}</main>{% include "templates/status.html" %}</body></html>`,
	)
	writeProjectFile(t, projectRoot, "pages/site.js", devPageSource())

	process := startDevProcess(t, projectRoot)
	defer process.stop(t)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	process.requireStarted(t, baseURL)

	body := requireHTTPBodyContains(t, baseURL, "Initial")
	require.Contains(t, body, "Initial Include")
	require.Contains(t, body, "fetch('/_veta/live'")
	requirePathMissing(t, filepath.Join(projectRoot, "dist"))

	revision := requireLiveRevision(t, baseURL+"_veta/live")

	writeProjectFile(t, projectRoot, "templates/status.html", "Updated Include")
	revision = requireLiveRevisionChanged(t, baseURL+"_veta/live", revision)

	includeBody := requireHTTPBodyContains(t, baseURL, "Updated Include")
	require.Contains(t, includeBody, "Initial")
	require.Contains(t, includeBody, "fetch('/_veta/live'")

	writeProjectFile(t, projectRoot, "content/message.txt", "Updated")
	revision = requireLiveRevisionChanged(t, baseURL+"_veta/live", revision)

	updatedBody := requireHTTPBodyContains(t, baseURL, "Updated")
	require.Contains(t, updatedBody, "fetch('/_veta/live'")
	requirePathMissing(t, filepath.Join(projectRoot, "dist"))

	started := time.Now()
	process.stop(t)
	require.Less(t, time.Since(started), 3*time.Second)
	require.NotContains(t, process.stderr.String(), "context deadline exceeded")
}

type devProcess struct {
	cancel  context.CancelFunc
	command *exec.Cmd
	done    chan struct{}
	stderr  *lineLog
	stdout  *lineLog
	waitErr error
}

// startDevProcess starts veta dev in a temporary project.
func startDevProcess(t *testing.T, projectRoot string) *devProcess {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	command := exec.CommandContext(
		ctx,
		vetaBinary,
		"dev",
	)
	command.Dir = projectRoot
	command.Env = isolatedEnvironment(t, projectRoot)

	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	stderr, err := command.StderrPipe()
	require.NoError(t, err)

	process := &devProcess{
		cancel:  cancel,
		command: command,
		done:    make(chan struct{}),
		stderr:  newLineLog(stderr),
		stdout:  newLineLog(stdout),
	}
	require.NoError(t, command.Start())
	go func() {
		defer close(process.done)
		process.waitErr = command.Wait()
	}()

	return process
}

// requireStarted waits until the dev process prints its local URL.
func (process *devProcess) requireStarted(t *testing.T, baseURL string) {
	t.Helper()

	deadline := time.After(15 * time.Second)
	for {
		select {
		case line, ok := <-process.stdout.lines:
			if !ok {
				t.Fatalf(
					"dev server stdout closed before startup\nstdout:\n%s\nstderr:\n%s",
					process.stdout.String(),
					process.stderr.String(),
				)
			}
			if strings.Contains(line, "Serving at "+baseURL) {
				return
			}
		case <-process.done:
			t.Fatalf(
				"veta dev exited before startup: %v\nstdout:\n%s\nstderr:\n%s",
				process.waitErr,
				process.stdout.String(),
				process.stderr.String(),
			)
		case <-deadline:
			t.Fatalf(
				"timed out waiting for dev server startup\nstdout:\n%s\nstderr:\n%s",
				process.stdout.String(),
				process.stderr.String(),
			)
		}
	}
}

// stop terminates the dev process and waits for it to exit.
func (process *devProcess) stop(t *testing.T) {
	t.Helper()

	select {
	case <-process.done:
		process.cancel()
		if runtime.GOOS != "windows" {
			require.NoError(
				t,
				process.waitErr,
				"stdout:\n%s\nstderr:\n%s",
				process.stdout.String(),
				process.stderr.String(),
			)
		}
		return
	default:
	}

	if process.command.Process != nil {
		if runtime.GOOS == "windows" {
			process.cancel()
		} else {
			err := process.command.Process.Signal(os.Interrupt)
			if err != nil && !errors.Is(err, os.ErrProcessDone) {
				require.NoError(t, err)
			}
		}
	}

	select {
	case <-process.done:
		process.cancel()
		if runtime.GOOS != "windows" {
			require.NoError(
				t,
				process.waitErr,
				"stdout:\n%s\nstderr:\n%s",
				process.stdout.String(),
				process.stderr.String(),
			)
		}
	case <-time.After(3 * time.Second):
		process.cancel()
		select {
		case <-process.done:
		case <-time.After(5 * time.Second):
		}
		t.Fatalf(
			"veta dev did not stop within 3 seconds\nstdout:\n%s\nstderr:\n%s",
			process.stdout.String(),
			process.stderr.String(),
		)
	}
}

// requireLiveRevision reads the current build revision from the live endpoint.
func requireLiveRevision(t *testing.T, endpoint string) uint64 {
	t.Helper()

	body := requireHTTPBodyContains(t, endpoint, "revision")
	var payload struct {
		Revision uint64 `json:"revision"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &payload))
	return payload.Revision
}

// requireLiveRevisionChanged polls the live endpoint until its revision
// advances past previous, returning the new revision.
func requireLiveRevisionChanged(t *testing.T, endpoint string, previous uint64) uint64 {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		body, err := getHTTPBody(t, endpoint)
		if err == nil {
			var payload struct {
				Revision uint64 `json:"revision"`
			}
			if json.Unmarshal([]byte(body), &payload) == nil && payload.Revision != previous {
				return payload.Revision
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for live revision to advance past %d", previous)
	return 0
}

type lineLog struct {
	buffer bytes.Buffer
	lines  chan string
	mutex  sync.Mutex
}

// newLineLog scans reader into a channel while retaining lines for diagnostics.
func newLineLog(reader io.Reader) *lineLog {
	log := &lineLog{lines: make(chan string, 100)}
	go func() {
		defer close(log.lines)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			log.mutex.Lock()
			log.buffer.WriteString(line)
			log.buffer.WriteByte('\n')
			log.mutex.Unlock()
			log.lines <- line
		}
	}()

	return log
}

// String returns all lines captured so far.
func (log *lineLog) String() string {
	log.mutex.Lock()
	defer log.mutex.Unlock()

	return log.buffer.String()
}

// requireHTTPBodyContains polls a URL until the response contains want.
func requireHTTPBodyContains(t *testing.T, address, want string) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	var lastErr error
	var lastBody string
	for time.Now().Before(deadline) {
		body, err := getHTTPBody(t, address)
		if err == nil && strings.Contains(body, want) {
			return body
		}
		lastErr = err
		lastBody = body
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf(
		"timed out waiting for %s to contain %q: %v\nlast body:\n%s",
		address,
		want,
		lastErr,
		lastBody,
	)
	return ""
}

// getHTTPBody requests a URL and returns a successful response body.
func getHTTPBody(t *testing.T, address string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return string(content), fmt.Errorf("unexpected status %s", response.Status)
	}

	return string(content), nil
}

// freeTCPPort returns an available local TCP port for a dev process.
func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	return address.Port
}

// devPageSource returns a page generator that reads content watched through dev.watch.
func devPageSource() string {
	return `
export default function({ files }) {
  const message = files.readFile("content/message.txt").trim();
  return [{ permalink: "/", template: "base", content: message }];
}
`
}
