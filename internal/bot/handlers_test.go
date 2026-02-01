package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_ParseArgs(t *testing.T) {
	t.Parallel()

	h := NewHandlers(nil)

	assert.Nil(t, h.parseArgs("/rate"))
	assert.Equal(t, []string{"USD", "VES"}, h.parseArgs("/rate USD VES"))
	assert.Equal(t, []string{"USD"}, h.parseArgs("/rate   USD"))
}

func TestHandler_CommandName(t *testing.T) {
	t.Parallel()

	h := NewHandlers(nil)

	assert.Equal(t, "/start", h.commandName("/start@ChiguiBot"))
	assert.Equal(t, "/start", h.commandName("/START extra"))
	assert.Equal(t, "", h.commandName(""))
}

func TestHandler_LanguageForCommand(t *testing.T) {
	t.Parallel()

	h := NewHandlers(nil)

	assert.Equal(t, LanguageEN, h.languageForCommand("/start"))
	assert.Equal(t, LanguageEN, h.languageForCommand("/help@bot"))
	assert.Equal(t, LanguageEN, h.languageForCommand("/rate USD VES"))
	assert.Equal(t, LanguageES, h.languageForCommand("/tasa USD VES"))
	assert.Equal(t, LanguageES, h.languageForCommand("/whatever"))
}
