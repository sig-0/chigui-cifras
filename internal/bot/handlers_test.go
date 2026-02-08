package bot

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_ParseArgs(t *testing.T) {
	t.Parallel()

	h := NewHandlers(nil, slog.Default(), nil)

	assert.Nil(t, h.parseArgs("/tasa"))
	assert.Equal(t, []string{"USD", "VES"}, h.parseArgs("/tasa USD VES"))
	assert.Equal(t, []string{"USD"}, h.parseArgs("/tasa   USD"))
}
