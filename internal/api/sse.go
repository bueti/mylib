package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

// registerSSE wires the scan-events stream. It's a plain chi handler
// because Huma doesn't model long-lived streaming responses.
func registerSSE(r chi.Router, d Deps) {
	r.Get("/api/scan/{id}/events", func(w http.ResponseWriter, req *http.Request) {
		if UserFromContext(req.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id := intParam(req, "id")
		if id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		// Verify the job exists; also lets us emit an immediate frame
		// if the job finished before the client connected.
		job, err := d.Store.GetScanJob(req.Context(), id)
		if errors.Is(err, library.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// If the job already finished, emit a single frame and return.
		if job.FinishedAt != nil {
			writeSSE(w, flusher, "job", toScanJobDTO(job))
			return
		}

		// Subscribe to live updates from the scanner.
		ch, cancel := d.Scanner.Subscribe(id)
		defer cancel()

		// Keepalive ticker so proxies don't drop the connection.
		keepalive := time.NewTicker(15 * time.Second)
		defer keepalive.Stop()

		for {
			select {
			case <-req.Context().Done():
				return
			case snap, ok := <-ch:
				if !ok {
					// subscription closed by scanner — final job reload.
					if final, err := d.Store.GetScanJob(req.Context(), id); err == nil {
						writeSSE(w, flusher, "job", toScanJobDTO(final))
					}
					return
				}
				snap.ID = id
				writeSSE(w, flusher, "job", toScanJobDTO(&snap))
			case <-keepalive.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
	flusher.Flush()
}
