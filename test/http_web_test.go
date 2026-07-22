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
	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
	"github.com/HongXiangZuniga/mongo-agent/pkg/http/web"
)

// validSessionID es un session ID que el fakeWebSessionManager por defecto
// acepta como sesión activa. Simula el ID opaco que Login guardó en Redis.
const validSessionID = "valid-session-id"

// createdSessionID es el ID que devuelve Create en el fake (simula el ID recién
// generado en un login correcto).
const createdSessionID = "created-session-id"

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

// fakeWebAuthenticator implementa auth.Authenticator para los tests web.
// Acepta la pareja usuario/contraseña configurada; para cualquier otra
// devuelve auth.ErrInvalidCredentials.
type fakeWebAuthenticator struct {
	okUser string
	okPass string
}

func (f fakeWebAuthenticator) Authenticate(ctx context.Context, username, password string) (auth.User, error) {
	if username == f.okUser && password == f.okPass {
		return auth.User{Username: username, PasswordHash: "hash"}, nil
	}
	return auth.User{}, auth.ErrInvalidCredentials
}

// fakeWebSessionManager implementa auth.WebSessionManager para los tests web.
// Captura el username recibido en Create, devuelve un ID fijo conocido, permite
// configurar el error de Validate (o una lista de IDs válidos) y captura el
// sessionID recibido en Revoke.
type fakeWebSessionManager struct {
	createID       string
	createErr      error
	createUsername string
	createCalls    int

	validIDs map[string]bool

	revokedID   string
	revokeCalls int
}

func newFakeWebSessionManager() *fakeWebSessionManager {
	return &fakeWebSessionManager{
		createID: createdSessionID,
		validIDs: map[string]bool{validSessionID: true},
	}
}

func (f *fakeWebSessionManager) Create(ctx context.Context, username string) (string, error) {
	f.createCalls++
	f.createUsername = username
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.createID, nil
}

func (f *fakeWebSessionManager) Validate(ctx context.Context, sessionID string) (auth.WebSession, error) {
	if f.validIDs[sessionID] {
		return auth.WebSession{Username: "admin", CreatedAt: time.Now()}, nil
	}
	return auth.WebSession{}, auth.ErrWebSessionNotFound
}

func (f *fakeWebSessionManager) Revoke(ctx context.Context, sessionID string) error {
	f.revokeCalls++
	f.revokedID = sessionID
	return nil
}

func setupWebHandler(fakeSvc *fakeAgentServiceWeb) *gin.Engine {
	return setupWebHandlerWithDeps(fakeSvc, fakeWebAuthenticator{okUser: "admin", okPass: "admin123"}, newFakeWebSessionManager())
}

func setupWebHandlerWithAuth(fakeSvc *fakeAgentServiceWeb, authr auth.Authenticator) *gin.Engine {
	return setupWebHandlerWithDeps(fakeSvc, authr, newFakeWebSessionManager())
}

func setupWebHandlerWithDeps(fakeSvc *fakeAgentServiceWeb, authr auth.Authenticator, mgr auth.WebSessionManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := web.CookieConfig{
		CookieName: "web_auth",
		MaxAge:     time.Hour,
		Secure:     false,
	}
	web.RegisterRoutes(r, web.NewWebHandler(fakeSvc, cfg, nil, authr, mgr), cfg, mgr)
	return r
}

func addAuthCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: validSessionID})
}

func TestCookieAuthMiddleware_RejectsMissingCookie(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))
}

func TestCookieAuth_ValidSessionAllows(t *testing.T) {
	fakeSvc := newFakeAgentServiceWeb()
	r := setupWebHandler(fakeSvc)

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	addAuthCookie(req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCookieAuth_InvalidSessionRedirects(t *testing.T) {
	// Petición normal: Validate falla → 303 a /web/login.
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: "no-registrada"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))

	// Petición htmx: Validate falla → HX-Redirect a /web/login.
	req2, _ := http.NewRequest(http.MethodGet, "/web/tabs/abc", nil)
	req2.AddCookie(&http.Cookie{Name: "web_auth", Value: "no-registrada"})
	req2.Header.Set("HX-Request", "true")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	assert.Equal(t, "/web/login", rec2.Header().Get("HX-Redirect"))
	assert.Empty(t, rec2.Body.String())
}

// TestCookieAuth_ArbitraryCookieRejected es el test de regresión del bug: una
// cookie cuyo valor es un API_TOKEN de ejemplo (o cualquier string no
// registrado como sesión) debe ser RECHAZADA por el middleware. Confirma que ya
// no basta con conocer API_TOKEN para entrar a /web.
func TestCookieAuth_ArbitraryCookieRejected(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	req, _ := http.NewRequest(http.MethodGet, "/web", nil)
	// Valor que en el modelo antiguo (bug) habría sido aceptado: el API_TOKEN.
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: "super-secret-api-token"})
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

func TestLogin_ValidCredentialsCreatesSessionAndSetsCookie(t *testing.T) {
	mgr := newFakeWebSessionManager()
	r := setupWebHandlerWithDeps(newFakeAgentServiceWeb(), fakeWebAuthenticator{okUser: "admin", okPass: "admin123"}, mgr)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin123")
	req, _ := http.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web", rec.Header().Get("Location"))

	// Create se invocó con el username autenticado.
	assert.Equal(t, 1, mgr.createCalls)
	assert.Equal(t, "admin", mgr.createUsername)

	setCookie := rec.Header().Get("Set-Cookie")
	// La cookie transporta el session ID devuelto por Create, NO el API_TOKEN
	// ni la contraseña.
	assert.Contains(t, setCookie, "web_auth="+createdSessionID)
	assert.NotContains(t, setCookie, "admin123")
	assert.Contains(t, setCookie, "HttpOnly")
	assert.Contains(t, setCookie, "Path=/web")
	assert.Contains(t, setCookie, "SameSite=Strict")
}

func TestLogin_InvalidCredentialsReturns401(t *testing.T) {
	mgr := newFakeWebSessionManager()
	r := setupWebHandlerWithDeps(newFakeAgentServiceWeb(), fakeWebAuthenticator{okUser: "admin", okPass: "admin123"}, mgr)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "wrong-password")
	req, _ := http.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Usuario o contraseña inválidos")
	assert.Empty(t, rec.Header().Get("Set-Cookie"))
	// No se crea sesión cuando las credenciales fallan.
	assert.Equal(t, 0, mgr.createCalls)
}

func TestLogin_UnknownUserReturns401WithGenericError(t *testing.T) {
	r := setupWebHandler(newFakeAgentServiceWeb())

	form := url.Values{}
	form.Set("username", "does-not-exist")
	form.Set("password", "whatever")
	req, _ := http.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	// Mismo mensaje genérico que en contraseña incorrecta: no revela cuál falló.
	assert.Contains(t, rec.Body.String(), "Usuario o contraseña inválidos")
	assert.Empty(t, rec.Header().Get("Set-Cookie"))
}

func TestLogout_RevokesServerSession(t *testing.T) {
	mgr := newFakeWebSessionManager()
	r := setupWebHandlerWithDeps(newFakeAgentServiceWeb(), fakeWebAuthenticator{okUser: "admin", okPass: "admin123"}, mgr)

	req, _ := http.NewRequest(http.MethodPost, "/web/logout", nil)
	req.AddCookie(&http.Cookie{Name: "web_auth", Value: validSessionID})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/web/login", rec.Header().Get("Location"))

	// La sesión se revocó del lado servidor con el ID de la cookie.
	assert.Equal(t, 1, mgr.revokeCalls)
	assert.Equal(t, validSessionID, mgr.revokedID)

	// Y la cookie del navegador se expira.
	parsedCookies := rec.Result().Cookies()
	require.NotEmpty(t, parsedCookies)
	cookie := parsedCookies[0]
	assert.Equal(t, "web_auth", cookie.Name)
	assert.True(t, cookie.MaxAge < 0 || cookie.Value == "")
	setCookie := rec.Header().Get("Set-Cookie")
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
