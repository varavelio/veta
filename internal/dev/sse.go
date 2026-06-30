package dev

import (
	"fmt"
	"net/http"
	"sync"
)

const reloadEvent = "reload"

type broadcaster struct {
	clients map[chan string]struct{}
	mutex   sync.Mutex
}

// newBroadcaster creates a Server-Sent Events broadcaster.
func newBroadcaster() *broadcaster {
	return &broadcaster{clients: map[chan string]struct{}{}}
}

// ServeHTTP streams reload events to a connected browser.
func (broadcaster *broadcaster) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("X-Accel-Buffering", "no")

	client := broadcaster.register()
	defer broadcaster.unregister(client)

	if _, err := fmt.Fprint(writer, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case event := <-client:
			if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, event); err != nil {
				return
			}
			flusher.Flush()
		case <-request.Context().Done():
			return
		}
	}
}

// broadcastReload sends a reload event to all connected browsers.
func (broadcaster *broadcaster) broadcastReload() {
	broadcaster.broadcast(reloadEvent)
}

// broadcast sends one event to every connected client without blocking rebuilds.
func (broadcaster *broadcaster) broadcast(event string) {
	broadcaster.mutex.Lock()
	defer broadcaster.mutex.Unlock()

	for client := range broadcaster.clients {
		select {
		case client <- event:
		default:
		}
	}
}

// register adds one browser connection to the broadcaster.
func (broadcaster *broadcaster) register() chan string {
	client := make(chan string, 1)

	broadcaster.mutex.Lock()
	broadcaster.clients[client] = struct{}{}
	broadcaster.mutex.Unlock()

	return client
}

// unregister removes one browser connection from the broadcaster.
func (broadcaster *broadcaster) unregister(client chan string) {
	broadcaster.mutex.Lock()
	delete(broadcaster.clients, client)
	broadcaster.mutex.Unlock()
	close(client)
}

// clientCount returns the current number of browser connections.
func (broadcaster *broadcaster) clientCount() int {
	broadcaster.mutex.Lock()
	defer broadcaster.mutex.Unlock()

	return len(broadcaster.clients)
}
