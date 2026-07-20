// Implementa los tests HTTP del adapter web.
package test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/mongo-agent/pkg/agent"
	"github.com/HongXiangZuniga/mongo-agent/pkg/http/web"
)

// fakeAgentServiceWeb implementa agent.AgentService para los tests web.
type fakeAgentServiceWeb struct {
	sessions             []agent.SessionSummary
	messages             map[string][]agent.Message
	askErr               error
	askCalls             int
	createCalls          int
	closeCalls           int
	listCalls            int
	getConvCalls         int
	lastAskQuestion      agent.Question
	lastCloseSession     string
	lastGetConversation  string
	collections          []string
	collectionsErr       error
	listCollectionsCalls int
}

func newFakeAgentServiceWeb() *fakeAgentServiceWeb {
	return &fakeAgentServiceWeb{
		sessions: []agent.SessionSummary{},
		messages: make(map[string][]agent.Message),
	}
}

func (f *fakeAgentServiceWeb) Ask(ctx context.Context, q agent.Question) (agent.Answer, error) {
	f.askCalls++
	f.lastAskQuestion = q

	if f.askErr != nil {
		return agent.Answer{}, f.askErr
	}

	f.messages[q.SessionID] = append(
		f.messages[q.SessionID],
		agent.Message{Role: agent.RoleUser, Content: q.Text},
		agent.Message{Role: agent.RoleAssistant, Content: "Respuesta simulada."},
	)

	return agent.Answer{SessionID: q.SessionID, Text: "Respuesta simulada."}, nil
}

func (f *fakeAgentServiceWeb) ListSessions(ctx context.Context) ([]agent.SessionSummary, error) {
	f.listCalls++
	return f.sessions, nil
}

func (f *fakeAgentServiceWeb) CreateSession(ctx context.Context) (agent.SessionSummary, error) {
	f.createCalls++
	return agent.SessionSummary{
		SessionID:    uuid.NewString(),
		Title:        "Nueva conversación",
		LastActivity: time.Now(),
	}, nil
}

func (f *fakeAgentServiceWeb) CloseSession(ctx context.Context, sessionID string) error {
	f.closeCalls++
	f.lastCloseSession = sessionID

	filtered := make([]agent.SessionSummary, 0, len(f.sessions))
	for _, s := range f.sessions {
		if s.SessionID != sessionID {
			filtered = append(filtered, s)
		}
	}
	f.sessions = filtered

	return nil
}

func (f *fakeAgentServiceWeb) GetConversation(ctx context.Context, sessionID string) ([]agent.Message, error) {
	f.getConvCalls++
	f.lastGetConversation = sessionID
	return f.messages[sessionID], nil
}

func (f *fakeAgentServiceWeb) ListAvailableCollections(ctx context.Context) ([]string, error) {
	f.listCollectionsCalls++
	if f.collectionsErr != nil {
		return nil, f.collectionsErr
	}
	return f.collections, nil
}

func setupWebHandler(fakeSvc *fakeAgentServiceWeb) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := web.CookieConfig{
		CookieName: "web_auth",
		MaxAge:     time.Hour,
		Secure:     false,
		APIToken:   "test-token",
	}
	web.RegisterRoutes(r, web.NewWebHandler(fakeSvc, cfg, nil), cfg)
	return r
}

func addAuthCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: "test-token"})
}

func TestCookieAuthMiddleware_RejectsMissingCookie(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))
}

func TestCookieAuthMiddleware_RejectsWrongToken(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: "wrong-token"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))
}

func TestCookieAuthMiddleware_HXRequestGetsHXRedirectHeader(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web/tabs/abc", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("HX-Redirect"))
	assert.Empty(t, rec.Body.String())
}

func TestCookieAuthMiddleware_NormalRequestGetsSeeOtherRedirect(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))
}

func TestIndex_ShowsEphemeralTabWhenNoSessions(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Nueva conversación")
	assert.Equal(t, 1, fakeSvc.createCalls)
	assert.Equal(t, 0, fakeSvc.askCalls)
}

func TestIndex_EmptyChatShowsDiscoveredCollectionsAsHint(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.collections = []string{"cards", "users"}
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "cards, users")
	assert.Equal(t, 1, fakeSvc.listCollectionsCalls)
}

func TestIndex_EmptyChatFallsBackToGenericHintWhenDiscoveryFails(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.collectionsErr = errors.New("mongodb unavailable: mongodb+srv://FAKEUSER:FAKEPASS@example-cluster.test")
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Escribe algo para empezar la conversación.")
	assert.NotContains(t, body, "FAKEUSER:FAKEPASS")
}

func TestIndex_ShowsMostRecentSessionAsActive(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Reciente", LastActivity: time.Now()},
		{SessionID: "s2", Title: "Antigua", LastActivity: time.Now().Add(-time.Hour)},
	}
	fakeSvc.messages["s1"] = []agent.Message{
		{Role: agent.RoleUser, Content: "mensaje s1"},
		{Role: agent.RoleAssistant, Content: "respuesta s1"},
	}

	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "mensaje s1")
	assert.Contains(t, body, "respuesta s1")
	assert.Equal(t, "s1", fakeSvc.lastGetConversation)
}

func TestLoginForm_RendersWithoutError(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web/login", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `action="/web/login"`)
}

func TestLogin_ValidTokenSetsCookieAndRedirects(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	form := url.Values{}
	form.Set("token", "test-token")
	req, _ := http.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web", rec.Header().Get("Location"))
	setCookie := rec.Header().Get("Set-Cookie")
	assert.Contains(t, setCookie, "web_auth=test-token")
	assert.Contains(t, setCookie, "HttpOnly")
	assert.Contains(t, setCookie, "Path=/web")
	assert.Contains(t, setCookie, "SameSite=Strict")
}

func TestLogin_InvalidTokenRendersErrorWithoutCookie(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	form := url.Values{}
	form.Set("token", "wrong-token")
	req, _ := http.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Token inválido")
	setCookie := rec.Header().Get("Set-Cookie")
	assert.NotContains(t, setCookie, "wrong-token")
}

func TestLogout_ClearsCookieAndRedirects(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodPost, "/web/logout", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))

	setCookie := rec.Header().Get("Set-Cookie")
	parsedCookies := rec.Result().Cookies()
	require.NotEmpty(t, parsedCookies)
	cookie := parsedCookies[0]
	assert.Equal(t, "web_auth", cookie.Name)
	assert.True(t, cookie.MaxAge < 0 || cookie.Value == "")
	assert.Contains(t, setCookie, "web_auth=")
	assert.Contains(t, setCookie, "SameSite=Strict")
}

func TestNewTab_ReturnsChatPanelAndOOBTabBar(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodPost, "/web/tabs", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `id="chat-panel"`)
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

func TestNewTab_DoesNotCallSessionStoreWrites(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodPost, "/web/tabs", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, fakeSvc.askCalls)
	assert.Equal(t, 1, fakeSvc.createCalls)
}

func TestSwitchTab_ReturnsChatPanelForRequestedSession(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Uno", LastActivity: time.Now()},
		{SessionID: "s2", Title: "Dos", LastActivity: time.Now().Add(-time.Hour)},
	}
	fakeSvc.messages["s1"] = []agent.Message{
		{Role: agent.RoleUser, Content: "mensaje s1"},
	}

	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web/tabs/s1", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "mensaje s1")
	assert.Equal(t, "s1", fakeSvc.lastGetConversation)
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

func TestSwitchTab_UpdatesActiveTabInOOBTabBar(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Uno", LastActivity: time.Now()},
		{SessionID: "s2", Title: "Dos", LastActivity: time.Now().Add(-time.Hour)},
	}

	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web/tabs/s2", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `hx-swap-oob="true"`)

	dosIdx := strings.Index(body, ">Dos<")
	unoIdx := strings.Index(body, ">Uno<")
	require.NotEqual(t, -1, dosIdx)
	require.NotEqual(t, -1, unoIdx)
	dosButtonStart := strings.LastIndex(body[:dosIdx], "<button")
	unoButtonStart := strings.LastIndex(body[:unoIdx], "<button")
	assert.Contains(t, body[dosButtonStart:dosIdx], "tab-active")
	assert.NotContains(t, body[unoButtonStart:unoIdx], "tab-active")
}

func TestSwitchTab_InvalidSessionIDReturns400(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web/tabs/invalid!id", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSendMessage_Success_ReturnsUpdatedPanelAndOOBTabBar(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Uno", LastActivity: time.Now()},
	}

	r := setupWebHandler(fakeSvc)

	form := url.Values{}
	form.Set("message", "hola")
	req, _ := http.NewRequest(http.MethodPost, "/web/tabs/s1/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "s1", fakeSvc.lastAskQuestion.SessionID)
	assert.Equal(t, "hola", fakeSvc.lastAskQuestion.Text)
	body := rec.Body.String()
	assert.Contains(t, body, `id="chat-panel"`)
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

func TestSendMessage_EmptyMessage_DoesNotCallAsk(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Uno", LastActivity: time.Now()},
	}

	r := setupWebHandler(fakeSvc)

	form := url.Values{}
	form.Set("message", "   ")
	req, _ := http.NewRequest(http.MethodPost, "/web/tabs/s1/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, fakeSvc.askCalls)
	body := rec.Body.String()
	assert.Contains(t, body, "el mensaje no puede estar vacío")
}

func TestSendMessage_AgentError_RendersGenericErrorInline(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Uno", LastActivity: time.Now()},
	}
	fakeSvc.askErr = errors.New("database connection leak xyz")

	r := setupWebHandler(fakeSvc)

	form := url.Values{}
	form.Set("message", "hola")
	req, _ := http.NewRequest(http.MethodPost, "/web/tabs/s1/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "no se pudo procesar el mensaje")
	assert.NotContains(t, body, "database connection leak xyz")
}

func TestCloseTab_ActivatesNextMostRecentSession(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Reciente", LastActivity: time.Now()},
		{SessionID: "s2", Title: "Siguiente", LastActivity: time.Now().Add(-time.Hour)},
	}
	fakeSvc.messages["s2"] = []agent.Message{
		{Role: agent.RoleUser, Content: "mensaje s2"},
	}

	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodDelete, "/web/tabs/s1", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "s1", fakeSvc.lastCloseSession)
	body := rec.Body.String()
	assert.Contains(t, body, "mensaje s2")
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

func TestCloseTab_CreatesEphemeralTabWhenNoneRemain(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Única", LastActivity: time.Now()},
	}

	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodDelete, "/web/tabs/s1", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, fakeSvc.closeCalls)
	assert.Equal(t, 1, fakeSvc.createCalls)
	body := rec.Body.String()
	assert.Contains(t, body, "Nueva conversación")
	assert.Contains(t, body, `hx-swap-oob="true"`)
}

func TestHTMXAsset_ServesJavascriptWithContentType(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web/static/htmx.min.js", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/javascript")
}

func TestAppCSSAsset_ServesCSSWithContentType(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	// Petición sin cookie de autenticación: la ruta es pública.
	req, _ := http.NewRequest(http.MethodGet, "/web/static/app.css", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/css")
}

func TestFontAsset_ServesWoff2WithContentType(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	// Petición sin cookie de autenticación: la ruta es pública.
	req, _ := http.NewRequest(http.MethodGet, "/web/static/fonts/inter-400.woff2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "font/woff2", rec.Header().Get("Content-Type"))
}

func TestFontAsset_RejectsUnknownFilename(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	// Petición sin cookie de autenticación: la ruta es pública.
	req, _ := http.NewRequest(http.MethodGet, "/web/static/fonts/otra-cosa.woff2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestIndex_ChatPanelIncludesTypingIndicator(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `id="chat-shell"`)
	assert.Contains(t, body, `id="typing-indicator"`)
	assert.Contains(t, body, `hx-indicator="#typing-indicator"`)
}

func TestIndex_RendersMarkdownTableAsHTMLTable(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "Tablas", LastActivity: time.Now()},
	}
	fakeSvc.messages["s1"] = []agent.Message{
		{
			Role:    agent.RoleAssistant,
			Content: "Resultados:\n| sku | ventas |\n|---|---|\n| a-1 | 20 |",
		},
	}
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `<table class="md-table">`)
	assert.Contains(t, body, "<th>sku</th>")
	assert.Contains(t, body, "<td>a-1</td>")
	assert.Contains(t, body, "<td>20</td>")
}

func TestIndex_MessageContentIsHTMLEscapedInsideTable(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "XSS", LastActivity: time.Now()},
	}
	fakeSvc.messages["s1"] = []agent.Message{
		{
			Role:    agent.RoleAssistant,
			Content: "| payload |\n|---|\n| <script>alert(1)</script> |",
		},
	}
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

func TestIndex_RendersCSVFenceAsDownloadLink(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	fakeSvc.sessions = []agent.SessionSummary{
		{SessionID: "s1", Title: "CSV", LastActivity: time.Now()},
	}
	fakeSvc.messages["s1"] = []agent.Message{
		{
			Role:    agent.RoleAssistant,
			Content: "Aquí tienes el resultado en CSV:\n\n```csv\nname,total\na,10\n```\n\nListo.",
		},
	}
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.NotContains(t, body, "```csv")
	assert.Contains(t, body, `download="resultado.csv"`)
	assert.Contains(t, body, "data:text/csv;charset=utf-8;base64,")
}
