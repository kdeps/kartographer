package treesitter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callees(f Func) []string {
	var out []string
	for _, c := range f.Calls {
		out = append(out, c.Callee)
	}
	return out
}

func TestPythonFunctionsAndCalls(t *testing.T) {
	src := []byte("def helper():\n    return 1\n\ndef main():\n    helper()\n    obj.run()\n")
	funcs, err := Extract("a.py", src)
	require.NoError(t, err)
	require.Len(t, funcs, 2)
	assert.Equal(t, "helper", funcs[0].Name)
	assert.Equal(t, 1, funcs[0].Line)
	assert.Equal(t, "main", funcs[1].Name)
	assert.Equal(t, 4, funcs[1].Line)
	assert.Equal(t, []string{"helper", "run"}, callees(funcs[1]))
}

func TestJavaMethodsAndInvocations(t *testing.T) {
	src := []byte("class A {\n  void run() { helper(); this.stop(); }\n  void helper() {}\n}\n")
	funcs, err := Extract("A.java", src)
	require.NoError(t, err)
	require.Len(t, funcs, 2)
	assert.Equal(t, "run", funcs[0].Name)
	assert.Equal(t, []string{"helper", "stop"}, callees(funcs[0]))
}

func TestRustFunctionsAndPathCalls(t *testing.T) {
	src := []byte("fn leaf() -> i32 { 1 }\nfn top() -> i32 { crate::util::leaf() }\n")
	funcs, err := Extract("lib.rs", src)
	require.NoError(t, err)
	require.Len(t, funcs, 2)
	assert.Equal(t, []string{"leaf"}, callees(funcs[1]), "a path call keeps its last identifier")
}

func TestTypeScriptAndTSX(t *testing.T) {
	src := []byte("function a(): number { return b(); }\nfunction b(): number { return 1; }\n")
	funcs, err := Extract("a.ts", src)
	require.NoError(t, err)
	require.Len(t, funcs, 2)
	assert.Equal(t, []string{"b"}, callees(funcs[0]))

	jsx := []byte("const C = () => <div/>;\nfunction d(): number { return e(); }\nfunction e(): number { return 1; }\n")
	funcs, err = Extract("c.tsx", jsx)
	require.NoError(t, err)
	assert.Len(t, funcs, 2)
}

func TestUnknownExtensionGivesNothing(t *testing.T) {
	funcs, err := Extract("notes.txt", []byte("def x(): pass"))
	require.NoError(t, err)
	assert.Empty(t, funcs)
	assert.Equal(t, "", Language("notes.txt"))
	assert.Equal(t, "python", Language("a.PY"))
	assert.True(t, Supports("rust"))
	assert.False(t, Supports("cobol"))
}

func TestLastIdent(t *testing.T) {
	assert.Equal(t, "run", lastIdent("obj.run"))
	assert.Equal(t, "leaf", lastIdent("crate::util::leaf"))
	assert.Equal(t, "foo", lastIdent("foo"))
	assert.Equal(t, "", lastIdent(""))
}
