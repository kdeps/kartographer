package codegraph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeIfaceModule writes a module where Direct calls Dog.Speak directly and
// ViaIface calls it through the Speaker interface.
func writeIfaceModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/iface\n\ngo 1.21\n",
		"iface.go": `package iface

type Speaker interface{ Speak() string }

type Dog struct{}

func (Dog) Speak() string { return "woof" }

func Direct() string { return Dog{}.Speak() }

func ViaIface(s Speaker) string { return s.Speak() }

func Use() string { return ViaIface(Dog{}) }
`,
	}
	for name, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
	}
	return dir
}

// symbolEnding returns the one indexed symbol whose name ends with suffix.
func symbolEnding(t *testing.T, ix *Index, suffix string) string {
	t.Helper()
	var found []string
	for name := range ix.Symbols {
		if strings.HasSuffix(name, suffix) {
			found = append(found, name)
		}
	}
	require.Len(t, found, 1, "symbol ending %q", suffix)
	return found[0]
}

func TestConfidenceMarksInterfaceCallsInferred(t *testing.T) {
	ix, err := Build(writeIfaceModule(t))
	require.NoError(t, err)

	speak := symbolEnding(t, ix, ".Dog.Speak")
	direct := symbolEnding(t, ix, ".Direct")
	viaIface := symbolEnding(t, ix, ".ViaIface")

	assert.Equal(t, "EXTRACTED", ix.Confidence(direct, speak), "a direct call site is extracted")
	assert.Equal(t, "INFERRED", ix.Confidence(viaIface, speak), "an interface call resolved by VTA is inferred")
	assert.Equal(t, "EXTRACTED", ix.Confidence(speak, direct), "an edge that is not in the graph defaults to extracted")
}

func TestConfidenceSurvivesTheCache(t *testing.T) {
	dir := writeIfaceModule(t)
	cache := t.TempDir()

	first, err := BuildCached(dir, cache)
	require.NoError(t, err)
	second, err := BuildCached(dir, cache)
	require.NoError(t, err)

	speak := symbolEnding(t, second, ".Dog.Speak")
	viaIface := symbolEnding(t, second, ".ViaIface")
	assert.Equal(t, first.Confidence(viaIface, speak), second.Confidence(viaIface, speak))
	assert.Equal(t, "INFERRED", second.Confidence(viaIface, speak), "the reloaded index keeps the tag")
}

func TestGraphMLCarriesConfidence(t *testing.T) {
	ix, err := Build(writeIfaceModule(t))
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, WriteGraphML(&b, ix))
	out := b.String()
	assert.Contains(t, out, `<key id="confidence" for="edge"`)
	assert.Contains(t, out, `<data key="confidence">INFERRED</data>`)
	assert.Contains(t, out, `<data key="confidence">EXTRACTED</data>`)
}

func TestCypherCarriesConfidence(t *testing.T) {
	ix, err := Build(writeIfaceModule(t))
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, WriteCypher(&b, ix))
	assert.Contains(t, b.String(), `[:CALLS {confidence: "INFERRED"}]`)
}

func TestCallersToolReportsConfidence(t *testing.T) {
	ix, err := Build(writeIfaceModule(t))
	require.NoError(t, err)

	var handle func(json.RawMessage) (string, error)
	for _, tool := range Tools(ix) {
		if tool.Name == "callers" {
			handle = tool.Handle
		}
	}
	require.NotNil(t, handle, "the callers tool is registered")

	raw, err := handle(json.RawMessage(`{"symbol":"Dog.Speak"}`))
	require.NoError(t, err)
	assert.Contains(t, raw, `"confidence":"INFERRED"`, "ViaIface reaches Speak through the interface")
	assert.Contains(t, raw, `"confidence":"EXTRACTED"`, "Direct reaches Speak directly")
}
