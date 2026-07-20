// Implementa los tests HTTP del adapter REST.
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/mongo-agent/pkg/agent"
	"github.com/HongXiangZuniga/mongo-agent/pkg/http/rest"
	"github.com/HongXiangZuniga/mongo-agent/pkg/utils"
)

// fakeAgentService implementa agent.AgentService para los tests HTTP.
type fakeAgentService struct {
	answer agent.Answer
	err    error
	asked  bool
}

func (f *fakeAgentService) Ask(ctx context.Context, q agent.Question) (agent.Answer, error) {
	f.asked = true
	if f.err != nil {
		return agent.Answer{}, f.err
	}
	return agent.Answer{
		SessionID: q.SessionID,
		Text:      "Respuesta simulada.",
	}, nil
}

func (f *fakeAgentService) ListSessions(ctx context.Context) ([]agent.SessionSummary, error) {
	return nil, nil
}

func (f *fakeAgentService) CreateSession(ctx context.Context) (agent.SessionSummary, error) {
	return agent.SessionSummary{}, nil
}

func (f *fakeAgentService) CloseSession(ctx context.Context, sessionID string) error {
	return nil
}

func (f *fakeAgentService) GetConversation(ctx context.Context, sessionID string) ([]agent.Message, error) {
	return nil, nil
}

func (f *fakeAgentService) ListAvailableCollections(ctx context.Context) ([]string, error) {
	return nil, nil
}

func setupHandler(svc agent.AgentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return rest.NewHandler(rest.NewAgentHandler(svc, nil), "test-token")
}

func TestAskQuestion_MissingAuthHeader(t *testing.T) {
	r := setupHandler(&fakeAgentService{})

	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString("{}"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAskQuestion_InvalidToken(t *testing.T) {
	r := setupHandler(&fakeAgentService{})

	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "wrong-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAskQuestion_MissingQuestionField(t *testing.T) {
	r := setupHandler(&fakeAgentService{})

	body := `{"session_id":"abc"}`
	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAskQuestion_Success(t *testing.T) {
	r := setupHandler(&fakeAgentService{})

	body := `{"session_id":"abc","question":"hola"}`
	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp rest.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "abc", data["session_id"])
	assert.Equal(t, "Respuesta simulada.", data["answer"])
}

func TestAskQuestion_GeneratesSessionIDWhenMissing(t *testing.T) {
	r := setupHandler(&fakeAgentService{})

	body := `{"question":"hola"}`
	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp rest.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["session_id"])
}

func TestAskQuestion_LLMErrorReturns502(t *testing.T) {
	svc := &fakeAgentService{
		err: utils.ErrLLMProviderUnavailable(errors.New("boom")),
	}
	r := setupHandler(svc)

	body := `{"session_id":"abc","question":"hola"}`
	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	assert.True(t, svc.asked)

	var resp rest.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusBadGateway, resp.Code)
	assert.Equal(t, "llm provider unavailable", resp.Message)
}

func TestAskQuestion_ErrorLogDoesNotContainSecret(t *testing.T) {
	svc := &fakeAgentService{
		err: errors.New("llm provider unavailable: request failed with header Authorization: Bearer supersecrettoken123"),
	}
	gin.SetMode(gin.TestMode)
	r := rest.NewHandler(rest.NewAgentHandler(svc, utils.NewSecretScrubber("supersecrettoken123")), "test-token")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	body := `{"session_id":"abc","question":"hola"}`
	req, _ := http.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.NotContains(t, buf.String(), "supersecrettoken123")
	assert.Contains(t, buf.String(), "[REDACTED]")
}
