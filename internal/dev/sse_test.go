package dev

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBroadcasterRegistersAndBroadcastsReload(t *testing.T) {
	broadcaster := newBroadcaster()
	client := broadcaster.register()
	require.Equal(t, 1, broadcaster.clientCount())

	broadcaster.broadcastReload()

	select {
	case event := <-client:
		require.Equal(t, reloadEvent, event)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reload event")
	}

	broadcaster.unregister(client)
	require.Equal(t, 0, broadcaster.clientCount())
}

func TestBroadcasterStreamsServerSentEvents(t *testing.T) {
	broadcaster := newBroadcaster()
	server := httptest.NewServer(broadcaster)
	defer server.Close()

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.URL,
		nil,
	)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	broadcaster.broadcastReload()
	requireEventuallyLine(t, lines, "event: reload")
	requireEventuallyLine(t, lines, "data: reload")
}

func requireEventuallyLine(t *testing.T, lines <-chan string, want string) {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("SSE stream closed before line %q", want)
			}
			if line == want {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for SSE line %q", want)
		}
	}
}
