package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type handlerStub struct {
	called     map[string]int
	listParams ListRecordsParams
	pullParams PullSyncParams
	lastID     string
}

func (h *handlerStub) Login(c *gin.Context)        { h.bump("login"); c.Status(http.StatusNoContent) }
func (h *handlerStub) Register(c *gin.Context)     { h.bump("register"); c.Status(http.StatusNoContent) }
func (h *handlerStub) Validate(c *gin.Context)     { h.bump("validate"); c.Status(http.StatusNoContent) }
func (h *handlerStub) UpsertRecord(c *gin.Context) { h.bump("upsert"); c.Status(http.StatusNoContent) }
func (h *handlerStub) PushSync(c *gin.Context)     { h.bump("push"); c.Status(http.StatusNoContent) }
func (h *handlerStub) ListRecords(c *gin.Context, params ListRecordsParams) {
	h.bump("list")
	h.listParams = params
	c.Status(http.StatusNoContent)
}
func (h *handlerStub) PullSync(c *gin.Context, params PullSyncParams) {
	h.bump("pull")
	h.pullParams = params
	c.Status(http.StatusNoContent)
}
func (h *handlerStub) DeleteRecord(c *gin.Context, id string) {
	h.bump("delete")
	h.lastID = id
	c.Status(http.StatusNoContent)
}
func (h *handlerStub) GetRecord(c *gin.Context, id string) {
	h.bump("get")
	h.lastID = id
	c.Status(http.StatusNoContent)
}

func (h *handlerStub) bump(name string) {
	if h.called == nil {
		h.called = map[string]int{}
	}
	h.called[name]++
}

func TestWrapperMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &handlerStub{}
	mwCalled := 0
	wrapper := ServerInterfaceWrapper{
		Handler: handler,
		HandlerMiddlewares: []MiddlewareFunc{
			func(c *gin.Context) {
				mwCalled++
			},
		},
		ErrorHandler: func(c *gin.Context, err error, status int) {
			c.Status(status)
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	wrapper.Login(ctx)
	if handler.called["login"] != 1 {
		t.Fatalf("expected login call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/register", nil)
	wrapper.Register(ctx)
	if handler.called["register"] != 1 {
		t.Fatalf("expected register call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/auth/validate", nil)
	wrapper.Validate(ctx)
	if handler.called["validate"] != 1 {
		t.Fatalf("expected validate call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/records?type=text&tag=t&query=q&limit=2&offset=1", nil)
	wrapper.ListRecords(ctx)
	if handler.called["list"] != 1 || handler.listParams.Type == nil {
		t.Fatalf("expected list call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/records", nil)
	wrapper.UpsertRecord(ctx)
	if handler.called["upsert"] != 1 {
		t.Fatalf("expected upsert call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Params = gin.Params{{Key: "id", Value: "r1"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/v1/records/r1", nil)
	wrapper.DeleteRecord(ctx)
	if handler.called["delete"] != 1 || handler.lastID != "r1" {
		t.Fatalf("expected delete call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Params = gin.Params{{Key: "id", Value: "r2"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/records/r2", nil)
	wrapper.GetRecord(ctx)
	if handler.called["get"] != 1 || handler.lastID != "r2" {
		t.Fatalf("expected get call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/sync?cursor=1&limit=10", nil)
	wrapper.PullSync(ctx)
	if handler.called["pull"] != 1 || handler.pullParams.Cursor == nil {
		t.Fatalf("expected pull call")
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/sync", nil)
	wrapper.PushSync(ctx)
	if handler.called["push"] != 1 {
		t.Fatalf("expected push call")
	}
	if mwCalled == 0 {
		t.Fatalf("expected middleware calls")
	}
}

func TestWrapperMiddlewareAbort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &handlerStub{}
	wrapper := ServerInterfaceWrapper{
		Handler: handler,
		HandlerMiddlewares: []MiddlewareFunc{
			func(c *gin.Context) {
				c.AbortWithStatus(http.StatusTeapot)
			},
		},
		ErrorHandler: func(c *gin.Context, err error, status int) {
			c.Status(status)
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	wrapper.Login(ctx)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected teapot status")
	}
	if handler.called["login"] != 0 {
		t.Fatalf("expected handler not called")
	}
}

func TestRegisterHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &handlerStub{}
	engine := gin.New()
	RegisterHandlers(engine, handler)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/validate", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status")
	}
	if handler.called["validate"] != 1 {
		t.Fatalf("expected handler call")
	}
}

func TestPayloadMergeFunctions(t *testing.T) {
	var payload Payload
	if err := payload.MergeCredentialPayload(CredentialPayload{Kind: Credential, Login: "l", Password: "p"}); err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if err := payload.MergeTextPayload(TextPayload{Kind: Text, Text: "t"}); err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if err := payload.MergeBinaryPayload(BinaryPayload{Kind: Binary, Data: []byte("b")}); err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if err := payload.MergeBankCardPayload(BankCardPayload{Kind: BankCard, Cardholder: "c", Number: "1", ExpiresAt: "12/30", Cvv: "123"}); err != nil {
		t.Fatalf("merge error: %v", err)
	}
}
