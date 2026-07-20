// Implementa tests para utilidades compartidas.
package test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/HongXiangZuniga/agente-inaricards/pkg/utils"
)

func TestIsValidSessionID_AcceptsAlphanumericWithDashesAndUnderscores(t *testing.T) {
	assert.True(t, utils.IsValidSessionID("abc123"))
	assert.True(t, utils.IsValidSessionID("session_1"))
	assert.True(t, utils.IsValidSessionID("session-1"))
	assert.True(t, utils.IsValidSessionID("a-b_c_1"))
}

func TestIsValidSessionID_RejectsEmptyOrSpecialCharacters(t *testing.T) {
	assert.False(t, utils.IsValidSessionID(""))
	assert.False(t, utils.IsValidSessionID("session id"))
	assert.False(t, utils.IsValidSessionID("session/id"))
	assert.False(t, utils.IsValidSessionID("session:id"))
	assert.False(t, utils.IsValidSessionID("sesión"))
}

func TestHTTPStatusForError_MapsKnownErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{
			name:   "tool loop exceeded",
			err:    utils.ErrToolLoopExceeded(),
			status: http.StatusBadGateway,
			msg:    "agent failed to finalize a response",
		},
		{
			name:   "request timeout",
			err:    utils.ErrRequestTimeout(),
			status: http.StatusGatewayTimeout,
			msg:    "request timed out",
		},
		{
			name:   "llm provider unavailable",
			err:    utils.ErrLLMProviderUnavailable(errors.New("boom")),
			status: http.StatusBadGateway,
			msg:    "llm provider unavailable",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := utils.HTTPStatusForError(tc.err)
			assert.Equal(t, tc.status, status)
			assert.Equal(t, tc.msg, msg)
		})
	}
}

func TestHTTPStatusForError_DefaultsToServiceUnavailable(t *testing.T) {
	status, msg := utils.HTTPStatusForError(errors.New("unexpected failure"))
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Equal(t, "service temporarily unavailable", msg)

	status, msg = utils.HTTPStatusForError(utils.ErrSessionStoreUnavailable(errors.New("redis down")))
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Equal(t, "service temporarily unavailable", msg)
}

func TestSecretScrubber_RedactsConfiguredSecrets(t *testing.T) {
	scrubber := utils.NewSecretScrubber("tok123", "key456")
	assert.Equal(t, "error: [REDACTED] failed with [REDACTED]", scrubber.Scrub("error: tok123 failed with key456"))
}

func TestSecretScrubber_LeavesTextWithoutSecretsUnchanged(t *testing.T) {
	scrubber := utils.NewSecretScrubber("tok123")
	assert.Equal(t, "plain error", scrubber.Scrub("plain error"))
}

func TestSecretScrubber_IgnoresEmptySecrets(t *testing.T) {
	scrubber := utils.NewSecretScrubber("", "abc")
	assert.Equal(t, "[REDACTED] and empty string stays", scrubber.Scrub("abc and empty string stays"))
}

func TestSecretScrubber_NilScrubberIsNoop(t *testing.T) {
	var scrubber *utils.SecretScrubber
	assert.Equal(t, "text", scrubber.Scrub("text"))
}

func TestRedactMongoCredentials_RedactsStandardURI(t *testing.T) {
	input := "failed to connect: mongodb+srv://dbuser:S3cr3t@cluster0.abcde.mongodb.net/?retryWrites=false"
	result := utils.RedactMongoCredentials(input)
	assert.NotContains(t, result, "dbuser")
	assert.NotContains(t, result, "S3cr3t")
	assert.Contains(t, result, "mongodb+srv://[REDACTED]:[REDACTED]@cluster0.abcde.mongodb.net")
}

func TestRedactMongoCredentials_RedactsGenericSchemeURI(t *testing.T) {
	input := "connecting to redis://user:pw@host:6379"
	result := utils.RedactMongoCredentials(input)
	assert.NotContains(t, result, "user:pw")
}

func TestRedactMongoCredentials_LeavesTextWithoutURIUnchanged(t *testing.T) {
	input := "collection not found"
	assert.Equal(t, input, utils.RedactMongoCredentials(input))
}

func TestRedactMongoCredentials_RedactsEvenWithoutExactURIMatch(t *testing.T) {
	input := "server selection error: mongodb://otheruser:otherpass@10.0.0.5:27017"
	result := utils.RedactMongoCredentials(input)
	assert.NotContains(t, result, "otheruser")
	assert.NotContains(t, result, "otherpass")
	assert.Contains(t, result, "[REDACTED]")
}

func TestSanitizeUserText_RemovesControlCharacters(t *testing.T) {
	input := "hola\x00mundo\x1b[31m"
	result := utils.SanitizeUserText(input)
	assert.NotContains(t, result, "\x00")
	assert.NotContains(t, result, "\x1b")
	assert.Contains(t, result, "holamundo")
}

func TestSanitizeUserText_PreservesNewlineAndTab(t *testing.T) {
	input := "linea1\nlinea2\ttab"
	assert.Equal(t, input, utils.SanitizeUserText(input))
}

func TestSanitizeUserText_PreservesUnicodePrintable(t *testing.T) {
	input := "¿Cuántas tarjetas hay? 🎉"
	assert.Equal(t, input, utils.SanitizeUserText(input))
}

func TestSanitizeUserText_AllControlCharsResultsInEmptyString(t *testing.T) {
	input := "\x00\x01\x02"
	assert.Equal(t, "", utils.SanitizeUserText(input))
}
