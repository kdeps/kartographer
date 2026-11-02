// Package mcp serves tools over the Model Context Protocol on stdio. It
// speaks newline-delimited JSON-RPC 2.0 and handles initialize, ping,
// tools/list and tools/call.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// ProtocolVersion is the MCP revision this server speaks.
const ProtocolVersion = "2024-11-05"

// Tool is one callable tool: its schema and its handler. The handler gets the
// raw JSON arguments and returns the text result.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handle      func(args json.RawMessage) (string, error)
}

// Server holds the tools it serves.
type Server struct {
	Name    string
	Version string
	Tools   []Tool
	// mu serialises requests in HTTP mode, where clients share the server: a
	// tool may write its notes, and nothing else is locked.
	mu sync.Mutex
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	errParse          = -32700
	errMethodNotFound = -32601
	errInvalidParams  = -32602
)

// Serve reads requests from in and writes responses to out until in closes.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	enc := json.NewEncoder(out)
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		resp, reply := s.handle(line)
		if !reply {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return sc.Err()
}

// handle returns the response for one line, and false for a notification.
func (s *Server) handle(line []byte) (response, bool) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return response{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: errParse, Message: err.Error()}}, true
	}
	isNotification := len(req.ID) == 0
	result, rerr := s.dispatch(req)
	if isNotification {
		return response{}, false
	}
	return response{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rerr}, true
}

func (s *Server) dispatch(req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.list()}, nil
	case "tools/call":
		return s.call(req.Params)
	default:
		return nil, &rpcError{Code: errMethodNotFound, Message: "unknown method " + req.Method}
	}
}

func (s *Server) list() []map[string]any {
	out := make([]map[string]any, 0, len(s.Tools))
	for _, t := range s.Tools {
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return out
}

func (s *Server) call(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errInvalidParams, Message: err.Error()}
	}
	for _, t := range s.Tools {
		if t.Name != p.Name {
			continue
		}
		args := p.Arguments
		if len(args) == 0 {
			args = json.RawMessage("{}")
		}
		text, err := t.Handle(args)
		if err != nil {
			return toolResult(fmt.Sprintf("error: %v", err), true), nil
		}
		return toolResult(text, false), nil
	}
	return nil, &rpcError{Code: errInvalidParams, Message: "unknown tool " + p.Name}
}

func toolResult(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isError,
	}
}
