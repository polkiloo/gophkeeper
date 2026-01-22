package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gophkeeper/internal/app"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
)

const (
	apiPrefix   = "/api"
	apiVersion  = "/v1"
	contentType = "application/json"
)

// Server exposes HTTP handlers for the API.
type Server struct {
	auth    inbound.AuthUseCase
	secrets inbound.SecretsUseCase
	sync    inbound.SyncUseCase
}

// NewServer wires HTTP transport with use cases.
func NewServer(auth inbound.AuthUseCase, secrets inbound.SecretsUseCase, syncUC inbound.SyncUseCase) *Server {
	return &Server{auth: auth, secrets: secrets, sync: syncUC}
}

func (s *Server) Register(c *gin.Context) {
	var req RegisterRequest
	if err := decodeJSON(c.Request, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	user, err := s.auth.Register(c.Request.Context(), inbound.RegisterInput{Login: req.Login, Password: req.Password})
	if err != nil {
		writeDomainError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, userToResponse(user))
}

func (s *Server) Login(c *gin.Context) {
	var req LoginRequest
	if err := decodeJSON(c.Request, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	session, err := s.auth.Login(c.Request.Context(), inbound.LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		writeDomainError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, sessionToResponse(session))
}

func (s *Server) Validate(c *gin.Context) {
	token, err := bearerToken(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	session, err := s.auth.Validate(c.Request.Context(), token)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, sessionToResponse(session))
}

func (s *Server) ListRecords(c *gin.Context, params ListRecordsParams) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	filter := inbound.RecordFilter{
		Type:           domain.RecordType(derefRecordType(params.Type)),
		Tag:            derefString(params.Tag),
		Query:          derefString(params.Query),
		Limit:          int(derefInt32(params.Limit)),
		Offset:         int(derefInt32(params.Offset)),
		IncludeDeleted: false,
	}

	records, err := s.secrets.List(ctx, filter)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	response := make([]Record, 0, len(records))
	for _, record := range records {
		dto, err := recordToOpenAPI(record)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err)
			return
		}
		response = append(response, dto)
	}

	writeJSON(c, http.StatusOK, response)
}

func (s *Server) UpsertRecord(c *gin.Context) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var req RecordUpsert
	if err := decodeJSON(c.Request, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	record, err := recordFromOpenAPI(req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	saved, err := s.secrets.Upsert(ctx, record)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	response, err := recordToOpenAPI(saved)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (s *Server) GetRecord(c *gin.Context, id string) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	record, err := s.secrets.Get(ctx, domain.RecordID(id))
	if err != nil {
		writeDomainError(c, err)
		return
	}

	response, err := recordToOpenAPI(record)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (s *Server) DeleteRecord(c *gin.Context, id string) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	if err := s.secrets.Delete(ctx, domain.RecordID(id)); err != nil {
		writeDomainError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) PullSync(c *gin.Context, params PullSyncParams) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	cursor := domain.SyncCursor(derefString(params.Cursor))
	limit := int(derefInt32(params.Limit))

	batch, err := s.sync.Pull(ctx, cursor, limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	changes := make([]RecordChange, 0, len(batch.Changes))
	for _, change := range batch.Changes {
		dto, err := changeToOpenAPI(change)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err)
			return
		}
		changes = append(changes, dto)
	}

	writeJSON(c, http.StatusOK, SyncBatch{Changes: changes, Cursor: string(batch.Cursor)})
}

func (s *Server) PushSync(c *gin.Context) {
	ctx, err := s.authenticate(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	var req SyncPush
	if err := decodeJSON(c.Request, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	changes := make([]domain.RecordChange, 0, len(req.Changes))
	for _, dto := range req.Changes {
		change, err := changeFromOpenAPI(dto)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
		changes = append(changes, change)
	}

	result, err := s.sync.Push(ctx, changes)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, SyncResult{Applied: int32(result.Applied), Rejected: int32(result.Rejected), Conflicts: idsToStrings(result.Conflicts)})
}

func (s *Server) authenticate(c *gin.Context) (context.Context, error) {
	token, err := bearerToken(c)
	if err != nil {
		return nil, err
	}
	session, err := s.auth.Validate(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}
	return app.WithUserID(c.Request.Context(), session.UserID), nil
}

func bearerToken(c *gin.Context) (string, error) {
	value := strings.TrimSpace(c.GetHeader("Authorization"))
	if value == "" {
		return "", domain.ErrUnauthorized
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return "", domain.ErrUnauthorized
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix)), nil
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(c *gin.Context, status int, value any) {
	c.Header("Content-Type", contentType)
	c.Status(status)
	_ = json.NewEncoder(c.Writer).Encode(value)
}

func writeError(c *gin.Context, status int, err error) {
	writeJSON(c, status, errorResponse{Error: err.Error()})
}

func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, err)
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, err)
	case errors.Is(err, domain.ErrConflict):
		writeError(c, http.StatusConflict, err)
	default:
		writeError(c, http.StatusInternalServerError, err)
	}
}

func idsToStrings(ids []domain.RecordID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, string(id))
	}
	return result
}

func derefInt32(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func derefRecordType(value *RecordType) RecordType {
	if value == nil {
		return ""
	}
	return *value
}
