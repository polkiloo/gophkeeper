package openapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestNewClientRequestEditor(t *testing.T) {
	var sawHeader bool
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Test") == "1" {
				sawHeader = true
			}
			return statusResponse(http.StatusOK), nil
		}),
	}

	client, err := NewClient("http://example.com/api", WithHTTPClient(httpClient), WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Test", "1")
		return nil
	}))
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	if _, err := client.Validate(context.Background()); err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if !sawHeader {
		t.Fatalf("expected request editor")
	}
	if client.Server[len(client.Server)-1] != '/' {
		t.Fatalf("expected trailing slash")
	}
}

func TestParseResponses(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseLoginResponse(resp); err == nil {
		t.Fatalf("expected login parse error")
	}

	plain := &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
	if _, err := ParseRegisterResponse(plain); err != nil {
		t.Fatalf("unexpected register parse error: %v", err)
	}
	if _, err := ParseDeleteRecordResponse(plain); err != nil {
		t.Fatalf("unexpected delete parse error: %v", err)
	}
}

func TestParseResponseErrors(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseRegisterResponse(resp); err == nil {
		t.Fatalf("expected register parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseValidateResponse(resp); err == nil {
		t.Fatalf("expected validate parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseListRecordsResponse(resp); err == nil {
		t.Fatalf("expected list parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseUpsertRecordResponse(resp); err == nil {
		t.Fatalf("expected upsert parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParseGetRecordResponse(resp); err == nil {
		t.Fatalf("expected get parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParsePullSyncResponse(resp); err == nil {
		t.Fatalf("expected pull parse error")
	}
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("{bad"))),
	}
	if _, err := ParsePushSyncResponse(resp); err == nil {
		t.Fatalf("expected push parse error")
	}
}

func TestResponseStatusAccessors(t *testing.T) {
	if (LoginResponse{}).StatusCode() != 0 {
		t.Fatalf("expected zero status")
	}
	if (ValidateResponse{}).StatusCode() != 0 {
		t.Fatalf("expected zero status")
	}
}

func TestRequestEditorError(t *testing.T) {
	expectedErr := errors.New("editor")
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return statusResponse(http.StatusOK), nil
		}),
	}
	client, err := NewClient("http://example.com/api", WithHTTPClient(httpClient), WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		return expectedErr
	}))
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	if _, err := client.Validate(context.Background()); err == nil {
		t.Fatalf("expected editor error")
	}
}
