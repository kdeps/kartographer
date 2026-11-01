// Package lsp is a minimal Language Server Protocol client over stdio. It
// covers the requests a code graph needs: initialize, document symbols and
// call hierarchy.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// Client is one running language server.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	r      *bufio.Reader
	nextID int
}

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Start runs the server command and returns a client connected to it.
func Start(bin string, args ...string) (*Client, error) {
	cmd := exec.Command(bin, args...) //nolint:gosec // the command comes from the registry
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Client{cmd: cmd, stdin: stdin, r: bufio.NewReader(stdout)}, nil
}

// Call sends a request and decodes its result into result.
func (c *Client) Call(method string, params, result any) error {
	c.nextID++
	id := json.RawMessage(strconv.Itoa(c.nextID))
	if err := c.write(message{JSONRPC: "2.0", ID: id, Method: method, Params: marshal(params)}); err != nil {
		return err
	}
	for {
		msg, err := c.read()
		if err != nil {
			return fmt.Errorf("lsp %s: %w", method, err)
		}
		if msg.Method != "" {
			// A server request or notification. Requests get a null reply so
			// the server does not wait on us.
			if len(msg.ID) > 0 {
				if err := c.write(message{JSONRPC: "2.0", ID: msg.ID, Result: json.RawMessage("null")}); err != nil {
					return err
				}
			}
			continue
		}
		if string(msg.ID) != string(id) {
			continue
		}
		if msg.Error != nil {
			return fmt.Errorf("lsp %s: %s", method, msg.Error.Message)
		}
		if result == nil || len(msg.Result) == 0 {
			return nil
		}
		return json.Unmarshal(msg.Result, result)
	}
}

// Notify sends a notification, which gets no reply.
func (c *Client) Notify(method string, params any) error {
	return c.write(message{JSONRPC: "2.0", Method: method, Params: marshal(params)})
}

// Close shuts the server down.
func (c *Client) Close() {
	_ = c.stdin.Close()
	_ = c.cmd.Wait()
}

func marshal(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func (c *Client) write(m message) error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

func (c *Client) read() (message, error) {
	length := -1
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return message{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, err = strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return message{}, err
			}
		}
	}
	if length < 0 {
		return message{}, errors.New("missing Content-Length header")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return message{}, err
	}
	var m message
	return m, json.Unmarshal(body, &m)
}
