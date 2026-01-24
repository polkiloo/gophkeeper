package openapi

import (
	"net/http"
	"testing"
)

func TestNewListRecordsRequest(t *testing.T) {
	req, err := NewListRecordsRequest("http://example.com/api", nil)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method")
	}

	valueType := RecordType("text")
	tag := "tag"
	query := "q"
	limit := int32(10)
	offset := int32(5)
	params := &ListRecordsParams{
		Type:   &valueType,
		Tag:    &tag,
		Query:  &query,
		Limit:  &limit,
		Offset: &offset,
	}
	req, err = NewListRecordsRequest("http://example.com/api", params)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if req.URL.RawQuery == "" {
		t.Fatalf("expected query params")
	}
}

func TestNewPullSyncRequest(t *testing.T) {
	req, err := NewPullSyncRequest("http://example.com/api", nil)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method")
	}

	cursor := "1"
	limit := int32(5)
	params := &PullSyncParams{Cursor: &cursor, Limit: &limit}
	req, err = NewPullSyncRequest("http://example.com/api", params)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if req.URL.RawQuery == "" {
		t.Fatalf("expected query params")
	}
}

func TestNewRequestErrors(t *testing.T) {
	if _, err := NewListRecordsRequest("://bad", nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := NewPullSyncRequest("://bad", nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := NewValidateRequest("://bad"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWithBaseURLSuccess(t *testing.T) {
	client, err := NewClient("http://example.com/api", WithBaseURL("http://example.com/base"))
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	if client.Server != "http://example.com/base/" {
		t.Fatalf("unexpected base url: %s", client.Server)
	}
}

func TestResponseStatusCodes(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusOK}
	if (LoginResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (RegisterResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (ValidateResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (ListRecordsResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (UpsertRecordResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (DeleteRecordResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (GetRecordResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (PullSyncResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
	if (PushSyncResponse{HTTPResponse: resp}).StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status code")
	}
}

func TestResponseStatusFallbacks(t *testing.T) {
	if (LoginResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected login status")
	}
	if (RegisterResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected register status")
	}
	if (ValidateResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected validate status")
	}
	if (ListRecordsResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected list status")
	}
	if (UpsertRecordResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected upsert status")
	}
	if (DeleteRecordResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected delete status")
	}
	if (GetRecordResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected get status")
	}
	if (PullSyncResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected pull status")
	}
	if (PushSyncResponse{}).Status() != http.StatusText(0) {
		t.Fatalf("unexpected push status")
	}

	if (LoginResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected login status code")
	}
	if (RegisterResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected register status code")
	}
	if (ValidateResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected validate status code")
	}
	if (ListRecordsResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected list status code")
	}
	if (UpsertRecordResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected upsert status code")
	}
	if (DeleteRecordResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected delete status code")
	}
	if (GetRecordResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected get status code")
	}
	if (PullSyncResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected pull status code")
	}
	if (PushSyncResponse{}).StatusCode() != 0 {
		t.Fatalf("unexpected push status code")
	}
}
