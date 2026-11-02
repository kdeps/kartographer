package mcp

import (
	"encoding/json"
	"io"
	"net/http"
)

// maxRequestBytes caps one JSON-RPC request body.
const maxRequestBytes = 16 << 20

// Handler serves the server over HTTP. A client POSTs one JSON-RPC request to
// the handler and reads the JSON response. A notification gets 202 Accepted and
// no body, as the MCP streamable HTTP transport asks.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "POST a JSON-RPC request", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body) > maxRequestBytes {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		s.mu.Lock()
		resp, reply := s.handle(body)
		s.mu.Unlock()
		if !reply {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
