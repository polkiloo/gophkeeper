package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gophkeeper/internal/ports/remote/openapi"
	"gopkg.in/yaml.v3"
)

// ClientFactory builds OpenAPI clients for a base URL.
type ClientFactory func(baseURL string) (*openapi.ClientWithResponses, error)

// Runner executes CLI commands.
type Runner struct {
	args    []string
	out     io.Writer
	errOut  io.Writer
	factory ClientFactory
	config  Config
	path    ConfigPath
}

// NewRunner constructs a CLI runner.
func NewRunner(args []string, stdout Stdout, stderr Stderr, factory ClientFactory, cfg Config, path ConfigPath) *Runner {
	return &Runner{args: args, out: io.Writer(stdout), errOut: io.Writer(stderr), factory: factory, config: cfg, path: path}
}

// Run executes the CLI flow.
func (r *Runner) Run(ctx context.Context) error {
	if len(r.args) < 2 {
		r.usage()
		return errors.New("command is required")
	}

	switch r.args[1] {
	case "auth":
		return r.handleAuth(ctx, r.args[2:])
	case "records":
		return r.handleRecords(ctx, r.args[2:])
	case "sync":
		return r.handleSync(ctx, r.args[2:])
	case "config":
		return r.handleConfig(ctx, r.args[2:])
	default:
		r.usage()
		return fmt.Errorf("unknown command: %s", r.args[1])
	}
}

func NewClientFactory(httpClient *http.Client) ClientFactory {
	return func(baseURL string) (*openapi.ClientWithResponses, error) {
		if httpClient == nil {
			return openapi.NewClientWithResponses(baseURL)
		}
		return openapi.NewClientWithResponses(baseURL, openapi.WithHTTPClient(httpClient))
	}
}

func (r *Runner) handleAuth(ctx context.Context, args []string) error {
	if len(args) < 1 {
		r.authUsage()
		return errors.New("auth command is required")
	}

	switch args[0] {
	case "register":
		fs := r.newFlagSet("auth register")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		login := fs.String("login", "", "User login")
		password := fs.String("password", "", "User password")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*login != "" && *password != "", "login and password are required"); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.RegisterWithResponse(ctx, openapi.RegisterRequest{Login: *login, Password: *password})
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	case "login":
		fs := r.newFlagSet("auth login")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		login := fs.String("login", "", "User login")
		password := fs.String("password", "", "User password")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*login != "" && *password != "", "login and password are required"); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.LoginWithResponse(ctx, openapi.LoginRequest{Login: *login, Password: *password})
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	case "validate":
		fs := r.newFlagSet("auth validate")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.ValidateWithResponse(ctx, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	default:
		r.authUsage()
		return fmt.Errorf("unknown auth command: %s", args[0])
	}
}

func (r *Runner) handleRecords(ctx context.Context, args []string) error {
	if len(args) < 1 {
		r.recordsUsage()
		return errors.New("records command is required")
	}

	switch args[0] {
	case "list":
		fs := r.newFlagSet("records list")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		recordType := fs.String("type", "", "Record type")
		tag := fs.String("tag", "", "Tag filter")
		query := fs.String("query", "", "Search query")
		limit := fs.Int("limit", 0, "Limit")
		offset := fs.Int("offset", 0, "Offset")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		params := &openapi.ListRecordsParams{}
		if *recordType != "" {
			value := openapi.RecordType(*recordType)
			params.Type = &value
		}
		if *tag != "" {
			params.Tag = tag
		}
		if *query != "" {
			params.Query = query
		}
		if *limit > 0 {
			value := int32(*limit)
			params.Limit = &value
		}
		if *offset > 0 {
			value := int32(*offset)
			params.Offset = &value
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.ListRecordsWithResponse(ctx, params, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	case "get":
		fs := r.newFlagSet("records get")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		id := fs.String("id", "", "Record ID")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*id != "", "id is required"); err != nil {
			return err
		}

		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.GetRecordWithResponse(ctx, *id, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	case "delete":
		fs := r.newFlagSet("records delete")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		id := fs.String("id", "", "Record ID")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*id != "", "id is required"); err != nil {
			return err
		}

		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.DeleteRecordWithResponse(ctx, *id, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, nil)
	case "upsert":
		fs := r.newFlagSet("records upsert")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		id := fs.String("id", "", "Record ID")
		recordType := fs.String("type", "", "Record type")
		version := fs.Int64("version", 0, "Record version")
		title := fs.String("title", "", "Meta title")
		description := fs.String("description", "", "Meta description")
		tags := fs.String("tags", "", "Comma-separated tags")
		attrs := fs.String("attrs", "", "Comma-separated key=value attributes")
		login := fs.String("login", "", "Credential login")
		password := fs.String("password", "", "Credential password")
		text := fs.String("text", "", "Text payload")
		data := fs.String("data", "", "Binary payload (base64)")
		cardholder := fs.String("cardholder", "", "Card holder")
		number := fs.String("number", "", "Card number")
		expiresAt := fs.String("expires-at", "", "Card expiry")
		cvv := fs.String("cvv", "", "Card CVV")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		if err := require(*recordType != "", "type is required"); err != nil {
			return err
		}
		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		payload, err := buildPayload(openapi.RecordType(*recordType), *login, *password, *text, *data, *cardholder, *number, *expiresAt, *cvv)
		if err != nil {
			return err
		}
		attrsMap, err := parseAttrs(*attrs)
		if err != nil {
			return err
		}
		meta := openapi.Metadata{
			Title:       stringPtr(*title),
			Description: stringPtr(*description),
			Tags:        stringSlicePtr(splitComma(*tags)),
			Attributes:  stringMapPtr(attrsMap),
		}

		req := openapi.RecordUpsert{
			Type:    openapi.RecordType(*recordType),
			Payload: payload,
			Meta:    meta,
		}
		if *id != "" {
			req.Id = id
		}
		if *version > 0 {
			req.Version = version
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.UpsertRecordWithResponse(ctx, req, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	default:
		r.recordsUsage()
		return fmt.Errorf("unknown records command: %s", args[0])
	}
}

func (r *Runner) handleSync(ctx context.Context, args []string) error {
	if len(args) < 1 {
		r.syncUsage()
		return errors.New("sync command is required")
	}

	switch args[0] {
	case "pull":
		fs := r.newFlagSet("sync pull")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		cursor := fs.String("cursor", "", "Sync cursor")
		limit := fs.Int("limit", 0, "Limit")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		params := &openapi.PullSyncParams{}
		if *cursor != "" {
			params.Cursor = cursor
		}
		if *limit > 0 {
			value := int32(*limit)
			params.Limit = &value
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.PullSyncWithResponse(ctx, params, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	case "push":
		fs := r.newFlagSet("sync push")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		token := fs.String("token", "", "Access token (or GOPHKEEPER_TOKEN)")
		file := fs.String("file", "", "Path to JSON payload")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*file != "", "file is required"); err != nil {
			return err
		}

		tokenValue := tokenOrEnv(*token)
		if err := require(tokenValue != "", "token is required"); err != nil {
			return err
		}

		data, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		var req openapi.SyncPush
		if err := json.Unmarshal(data, &req); err != nil {
			return err
		}

		client, err := r.factory(*baseURL)
		if err != nil {
			return err
		}
		resp, err := client.PushSyncWithResponse(ctx, req, r.bearerEditor(tokenValue))
		if err != nil {
			return err
		}
		return r.printResponse(resp.HTTPResponse, resp.Body, resp.JSON200)
	default:
		r.syncUsage()
		return fmt.Errorf("unknown sync command: %s", args[0])
	}
}

func (r *Runner) bearerEditor(token string) openapi.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if token == "" {
			return nil
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func (r *Runner) printResponse(resp *http.Response, raw []byte, body any) error {
	if resp == nil {
		return errors.New("empty response")
	}
	if resp.StatusCode >= 400 {
		if len(raw) == 0 {
			return fmt.Errorf("request failed: %s", resp.Status)
		}
		return fmt.Errorf("%s", string(raw))
	}
	if body == nil {
		fmt.Fprintf(r.out, "%s\n", resp.Status)
		return nil
	}
	output, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(r.out, string(output))
	return nil
}

func (r *Runner) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(r.errOut)
	return fs
}

func (r *Runner) usage() {
	fmt.Fprintln(r.errOut, "usage: gophkeeper-cli <auth|records|sync|config> <command> [flags]")
	fmt.Fprintln(r.errOut, "run with 'auth', 'records', 'sync', or 'config' to see available commands")
}

func (r *Runner) authUsage() {
	fmt.Fprintln(r.errOut, "usage: gophkeeper-cli auth <register|login|validate> [flags]")
}

func (r *Runner) recordsUsage() {
	fmt.Fprintln(r.errOut, "usage: gophkeeper-cli records <list|get|upsert|delete> [flags]")
}

func (r *Runner) syncUsage() {
	fmt.Fprintln(r.errOut, "usage: gophkeeper-cli sync <pull|push> [flags]")
}

func (r *Runner) configUsage() {
	fmt.Fprintln(r.errOut, "usage: gophkeeper-cli config <init> [flags]")
}

func (r *Runner) handleConfig(ctx context.Context, args []string) error {
	_ = ctx
	if len(args) < 1 {
		r.configUsage()
		return errors.New("config command is required")
	}

	switch args[0] {
	case "init":
		fs := r.newFlagSet("config init")
		defaultPath := r.resolveConfigPath()
		path := fs.String("path", defaultPath, "Config file path")
		baseURL := fs.String("base-url", r.config.BaseURL, "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := require(*baseURL != "", "base-url is required"); err != nil {
			return err
		}
		if err := r.writeConfig(*path, Config{BaseURL: *baseURL}); err != nil {
			return err
		}
		fmt.Fprintf(r.out, "config written to %s\n", *path)
		return nil
	default:
		r.configUsage()
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func buildPayload(recordType openapi.RecordType, login, password, text, data, cardholder, number, expiresAt, cvv string) (openapi.Payload, error) {
	var payload openapi.Payload
	switch recordType {
	case openapi.Credential:
		if err := require(login != "" && password != "", "login and password are required"); err != nil {
			return payload, err
		}
		return payload, payload.FromCredentialPayload(openapi.CredentialPayload{Kind: recordType, Login: login, Password: password})
	case openapi.Text:
		if err := require(text != "", "text is required"); err != nil {
			return payload, err
		}
		return payload, payload.FromTextPayload(openapi.TextPayload{Kind: recordType, Text: text})
	case openapi.Binary:
		decoded, err := decodeBase64(data)
		if err != nil {
			return payload, err
		}
		return payload, payload.FromBinaryPayload(openapi.BinaryPayload{Kind: recordType, Data: decoded})
	case openapi.BankCard:
		if err := require(cardholder != "" && number != "" && expiresAt != "" && cvv != "", "card details are required"); err != nil {
			return payload, err
		}
		return payload, payload.FromBankCardPayload(openapi.BankCardPayload{Kind: recordType, Cardholder: cardholder, Number: number, ExpiresAt: expiresAt, Cvv: cvv})
	default:
		return payload, fmt.Errorf("unknown record type: %s", recordType)
	}
}

func decodeBase64(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("data is required")
	}
	return base64.StdEncoding.DecodeString(value)
}

func tokenOrEnv(token string) string {
	if token != "" {
		return token
	}
	return os.Getenv("GOPHKEEPER_TOKEN")
}

func splitComma(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func parseAttrs(value string) (map[string]string, error) {
	if value == "" {
		return nil, nil
	}
	result := map[string]string{}
	for _, pair := range strings.Split(value, ",") {
		item := strings.TrimSpace(pair)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid attribute: %s", item)
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringSlicePtr(value []string) *[]string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func stringMapPtr(value map[string]string) *map[string]string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func require(ok bool, message string) error {
	if !ok {
		return errors.New(message)
	}
	return nil
}

func (r *Runner) resolveConfigPath() string {
	if r.path != "" {
		return string(r.path)
	}
	return DefaultConfigPath()
}

func (r *Runner) writeConfig(path string, cfg Config) error {
	if path == "" {
		return errors.New("config path is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
