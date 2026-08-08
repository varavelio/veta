package dev

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// revision tracks completed site builds so browsers can detect when to reload.
//
// The dev server avoids long-lived Server-Sent Events connections, which occupy
// a browser connection slot per open tab and can leave half-open connections
// behind. Instead, browsers poll the live endpoint and reload whenever the
// reported revision changes after a completed build.
type revision struct {
	value atomic.Uint64
}

// newRevision creates a live-reload revision counter.
func newRevision() *revision {
	return &revision{}
}

// bump advances the revision after one completed build.
func (revision *revision) bump() {
	revision.value.Add(1)
}

// current returns the latest completed build revision.
func (revision *revision) current() uint64 {
	return revision.value.Load()
}

// ServeHTTP reports the current build revision as JSON.
func (revision *revision) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(writer, `{"revision":%d}`, revision.current())
}
