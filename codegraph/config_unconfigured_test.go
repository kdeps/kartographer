package codegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureRejectsBadYAML(t *testing.T) {
	assert.Error(t, Configure([]byte("ranking: [unclosed")))
}
