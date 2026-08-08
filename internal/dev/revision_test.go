package dev

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRevisionBumpsAndReportsCurrent(t *testing.T) {
	revision := newRevision()
	require.Equal(t, uint64(0), revision.current())

	revision.bump()
	revision.bump()
	require.Equal(t, uint64(2), revision.current())
}

func TestLiveEndpointServesRevisionJSON(t *testing.T) {
	revision := newRevision()
	revision.bump()
	revision.bump()
	server := httptest.NewServer(revision)
	defer server.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "no-store", response.Header.Get("Cache-Control"))
	require.Equal(t, "application/json", response.Header.Get("Content-Type"))

	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"revision":2}`, string(content))
}

func TestLiveEndpointRejectsNonGetRequests(t *testing.T) {
	revision := newRevision()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/_veta/live", nil)

	revision.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	require.Equal(t, http.MethodGet, recorder.Header().Get("Allow"))
}
