package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer() *Server {
	return &Server{Name: "t", Version: "1", Tools: []Tool{
		{
			Name:        "echo",
			Description: "echo the text",
			InputSchema: map[string]any{"type": "object"},
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct{ Text string }
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Text == "" {
					return "", errors.New("text is required")
				}
				return a.Text, nil
			},
		},
	}}
}

func serve(t *testing.T, input string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, testServer().Serve(strings.NewReader(input), &out))
	var resps []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var r map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &r))
		resps = append(resps, r)
	}
	return resps
}

func TestInitializeListAndCall(t *testing.T) {
	resps := serve(t, strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	}, "\n"))
	require.Len(t, resps, 4, "the notification gets no response")

	init := resps[0]["result"].(map[string]any)
	assert.Equal(t, ProtocolVersion, init["protocolVersion"])

	tools := resps[1]["result"].(map[string]any)["tools"].([]any)
	assert.Equal(t, "echo", tools[0].(map[string]any)["name"])

	content := resps[2]["result"].(map[string]any)["content"].([]any)
	assert.Equal(t, "hi", content[0].(map[string]any)["text"])
	assert.Equal(t, false, resps[2]["result"].(map[string]any)["isError"])
}

func TestToolErrorsAreResults(t *testing.T) {
	resps := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	result := resps[0]["result"].(map[string]any)
	assert.Equal(t, true, result["isError"])
	assert.Contains(t, result["content"].([]any)[0].(map[string]any)["text"], "text is required")
}

func TestProtocolErrors(t *testing.T) {
	resps := serve(t, strings.Join([]string{
		`not json`,
		`{"jsonrpc":"2.0","id":2,"method":"nope"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"missing"}}`,
	}, "\n"))
	require.Len(t, resps, 3)
	assert.NotNil(t, resps[0]["error"])
	assert.Equal(t, float64(-32601), resps[1]["error"].(map[string]any)["code"])
	assert.Equal(t, float64(-32602), resps[2]["error"].(map[string]any)["code"])
}
