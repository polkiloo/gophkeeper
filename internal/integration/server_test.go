//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gophkeeper/internal/ports/remote/openapi"
)

func TestServerContainerAPI(t *testing.T) {
	ctx, client := startServer(t)
	token, userID := registerAndLogin(t, ctx, client, "alice", "secret")

	validateResp, err := client.ValidateWithResponse(ctx, authEditor(token))
	if err != nil {
		t.Fatalf("validate request failed: %v", err)
	}
	if validateResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected validate status: %d", validateResp.StatusCode())
	}

	meta := sampleMetadata()
	recordIDs := []string{}

	textPayload := mustTextPayload(t, "secret-data")
	textRecord := openapi.RecordUpsert{Type: openapi.Text, Payload: textPayload, Meta: meta}
	textResp := upsertRecord(t, ctx, client, token, textRecord)
	recordIDs = append(recordIDs, textResp.Id)

	credPayload := mustCredentialPayload(t, "login", "pass")
	credRecord := openapi.RecordUpsert{Type: openapi.Credential, Payload: credPayload, Meta: meta}
	credResp := upsertRecord(t, ctx, client, token, credRecord)
	recordIDs = append(recordIDs, credResp.Id)

	binaryPayload := mustBinaryPayload(t, []byte("binary"))
	binaryRecord := openapi.RecordUpsert{Type: openapi.Binary, Payload: binaryPayload, Meta: meta}
	binaryResp := upsertRecord(t, ctx, client, token, binaryRecord)
	recordIDs = append(recordIDs, binaryResp.Id)

	cardPayload := mustBankCardPayload(t, "User", "4111111111111111", "10/28", "123")
	cardRecord := openapi.RecordUpsert{Type: openapi.BankCard, Payload: cardPayload, Meta: meta}
	cardResp := upsertRecord(t, ctx, client, token, cardRecord)
	recordIDs = append(recordIDs, cardResp.Id)

	verifyRecord(t, ctx, client, token, textResp.Id, openapi.Text, meta)
	verifyRecord(t, ctx, client, token, credResp.Id, openapi.Credential, meta)
	verifyRecord(t, ctx, client, token, binaryResp.Id, openapi.Binary, meta)
	verifyRecord(t, ctx, client, token, cardResp.Id, openapi.BankCard, meta)

	listResp, err := client.ListRecordsWithResponse(ctx, &openapi.ListRecordsParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if listResp.StatusCode() != http.StatusOK || listResp.JSON200 == nil {
		t.Fatalf("unexpected list status: %d", listResp.StatusCode())
	}
	for _, id := range recordIDs {
		if !recordPresent(listResp.JSON200, id) {
			t.Fatalf("record not found in list: %s", id)
		}
	}

	pullResp, err := client.PullSyncWithResponse(ctx, &openapi.PullSyncParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("pull request failed: %v", err)
	}
	if pullResp.StatusCode() != http.StatusOK || pullResp.JSON200 == nil {
		t.Fatalf("unexpected pull status: %d", pullResp.StatusCode())
	}
	if len(pullResp.JSON200.Changes) == 0 {
		t.Fatalf("expected sync changes")
	}

	syncPayload := mustTextPayload(t, "sync-data")
	change := openapi.RecordChange{
		RecordId:   fmt.Sprintf("sync-%d", time.Now().UnixNano()),
		OwnerId:    userID,
		Type:       openapi.Text,
		Change:     openapi.Upsert,
		Payload:    &syncPayload,
		Meta:       &meta,
		Version:    1,
		HappenedAt: time.Now().UTC(),
	}
	pushResp, err := client.PushSyncWithResponse(ctx, openapi.SyncPush{Changes: []openapi.RecordChange{change}}, authEditor(token))
	if err != nil {
		t.Fatalf("push request failed: %v", err)
	}
	if pushResp.StatusCode() != http.StatusOK || pushResp.JSON200 == nil {
		t.Fatalf("unexpected push status: %d", pushResp.StatusCode())
	}
	if pushResp.JSON200.Applied != 1 {
		t.Fatalf("expected applied=1")
	}

	getSyncedResp, err := client.GetRecordWithResponse(ctx, change.RecordId, authEditor(token))
	if err != nil {
		t.Fatalf("get synced record failed: %v", err)
	}
	if getSyncedResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected get synced status: %d", getSyncedResp.StatusCode())
	}

	deleteResp, err := client.DeleteRecordWithResponse(ctx, textResp.Id, authEditor(token))
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if deleteResp.StatusCode() != http.StatusNoContent {
		t.Fatalf("unexpected delete status: %d", deleteResp.StatusCode())
	}

	getDeletedResp, err := client.GetRecordWithResponse(ctx, textResp.Id, authEditor(token))
	if err != nil {
		t.Fatalf("get deleted record failed: %v", err)
	}
	if getDeletedResp.StatusCode() != http.StatusNotFound {
		t.Fatalf("unexpected deleted get status: %d", getDeletedResp.StatusCode())
	}
}

func TestScenarioNewUserFlow(t *testing.T) {
	ctx, client := startServer(t)
	token, userID := registerAndLogin(t, ctx, client, "new-user", "secret")

	meta := sampleMetadata()
	payload := mustTextPayload(t, "hello")
	record := openapi.RecordUpsert{Type: openapi.Text, Payload: payload, Meta: meta}
	upserted := upsertRecord(t, ctx, client, token, record)

	pullResp, err := client.PullSyncWithResponse(ctx, &openapi.PullSyncParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("pull request failed: %v", err)
	}
	if pullResp.StatusCode() != http.StatusOK || pullResp.JSON200 == nil {
		t.Fatalf("unexpected pull status: %d", pullResp.StatusCode())
	}

	change, ok := findChange(pullResp.JSON200.Changes, upserted.Id)
	if !ok {
		t.Fatalf("expected change for record")
	}
	if change.Change != openapi.Upsert || change.Type != openapi.Text || change.OwnerId != userID {
		t.Fatalf("unexpected change data")
	}
	if change.Meta == nil {
		t.Fatalf("expected change metadata")
	}
	assertMetadata(t, *change.Meta, meta)
}

func TestScenarioExistingUserFlow(t *testing.T) {
	ctx, client := startServer(t)
	token, userID := registerAndLogin(t, ctx, client, "existing-user", "secret")

	meta := sampleMetadata()
	payload := mustTextPayload(t, "stored-data")
	record := openapi.RecordUpsert{Type: openapi.Text, Payload: payload, Meta: meta}
	upserted := upsertRecord(t, ctx, client, token, record)

	loginResp, err := client.LoginWithResponse(ctx, openapi.LoginRequest{Login: "existing-user", Password: "secret"})
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if loginResp.StatusCode() != http.StatusOK || loginResp.JSON200 == nil {
		t.Fatalf("unexpected login status: %d", loginResp.StatusCode())
	}
	if loginResp.JSON200.UserId != userID {
		t.Fatalf("unexpected user id")
	}
	token = loginResp.JSON200.Token

	pullResp, err := client.PullSyncWithResponse(ctx, &openapi.PullSyncParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("pull request failed: %v", err)
	}
	if pullResp.StatusCode() != http.StatusOK || pullResp.JSON200 == nil {
		t.Fatalf("unexpected pull status: %d", pullResp.StatusCode())
	}
	if _, ok := findChange(pullResp.JSON200.Changes, upserted.Id); !ok {
		t.Fatalf("expected change for record")
	}

	getResp, err := client.GetRecordWithResponse(ctx, upserted.Id, authEditor(token))
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if getResp.StatusCode() != http.StatusOK || getResp.JSON200 == nil {
		t.Fatalf("unexpected get status: %d", getResp.StatusCode())
	}
	assertMetadata(t, getResp.JSON200.Meta, meta)
	if getResp.JSON200.Id != upserted.Id {
		t.Fatalf("unexpected record id")
	}
}
