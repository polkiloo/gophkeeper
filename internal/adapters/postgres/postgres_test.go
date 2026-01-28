package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"

	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

var (
	driverOnce sync.Once
	testState  *fakeState
)

type fakeState struct {
	mu         sync.Mutex
	users      map[string]userRow
	records    map[string]recordRow
	changes    []changeRow
	nextChange int64
}

type userRow struct {
	id        string
	login     string
	hash      string
	createdAt time.Time
}

type recordRow struct {
	id        string
	ownerID   string
	rtype     string
	payload   []byte
	meta      []byte
	version   int64
	updatedAt time.Time
	deleted   bool
}

type changeRow struct {
	id         int64
	recordID   string
	ownerID    string
	rtype      string
	changeType string
	payload    []byte
	meta       []byte
	version    int64
	happenedAt time.Time
}

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConn{state: testState}, nil
}

type fakeConn struct {
	state *fakeState
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { return &fakeTx{}, nil }

func (c *fakeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch {
	case strings.Contains(query, "INSERT INTO users"):
		id := fmt.Sprint(args[0].Value)
		login := fmt.Sprint(args[1].Value)
		if _, exists := c.state.users[login]; exists {
			return nil, &pq.Error{Code: "23505"}
		}
		row := userRow{
			id:        id,
			login:     login,
			hash:      fmt.Sprint(args[2].Value),
			createdAt: args[3].Value.(time.Time),
		}
		if c.state.users == nil {
			c.state.users = make(map[string]userRow)
		}
		c.state.users[login] = row
		return fakeResult(1), nil
	case strings.Contains(query, "UPDATE users SET password_hash"):
		id := fmt.Sprint(args[0].Value)
		for login, row := range c.state.users {
			if row.id == id {
				row.hash = fmt.Sprint(args[1].Value)
				c.state.users[login] = row
				return fakeResult(1), nil
			}
		}
		return fakeResult(0), nil
	case strings.Contains(query, "INSERT INTO records"):
		row := recordRow{
			id:        fmt.Sprint(args[0].Value),
			ownerID:   fmt.Sprint(args[1].Value),
			rtype:     fmt.Sprint(args[2].Value),
			payload:   args[3].Value.([]byte),
			meta:      args[4].Value.([]byte),
			version:   args[5].Value.(int64),
			updatedAt: args[6].Value.(time.Time),
			deleted:   false,
		}
		if c.state.records == nil {
			c.state.records = make(map[string]recordRow)
		}
		c.state.records[row.id] = row
		return fakeResult(1), nil
	case strings.Contains(query, "UPDATE records SET deleted"):
		id := fmt.Sprint(args[0].Value)
		row, ok := c.state.records[id]
		if !ok || row.deleted {
			return fakeResult(0), nil
		}
		row.deleted = true
		c.state.records[id] = row
		return fakeResult(1), nil
	case strings.Contains(query, "INSERT INTO change_log"):
		if c.state.nextChange == 0 {
			c.state.nextChange = 1
		}
		row := changeRow{
			id:         c.state.nextChange,
			recordID:   fmt.Sprint(args[0].Value),
			ownerID:    fmt.Sprint(args[1].Value),
			rtype:      fmt.Sprint(args[2].Value),
			changeType: fmt.Sprint(args[3].Value),
			payload:    args[4].Value.([]byte),
			meta:       args[5].Value.([]byte),
			version:    args[6].Value.(int64),
			happenedAt: args[7].Value.(time.Time),
		}
		c.state.nextChange++
		c.state.changes = append(c.state.changes, row)
		return fakeResult(1), nil
	default:
		return fakeResult(0), nil
	}
}

func (c *fakeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch {
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE login"):
		login := fmt.Sprint(args[0].Value)
		row, ok := c.state.users[login]
		if !ok {
			return &fakeRows{columns: userColumns()}, nil
		}
		return &fakeRows{
			columns: userColumns(),
			values: [][]driver.Value{{
				row.id, row.login, row.hash, row.createdAt,
			}},
		}, nil
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE id"):
		id := fmt.Sprint(args[0].Value)
		for _, row := range c.state.users {
			if row.id == id {
				return &fakeRows{
					columns: userColumns(),
					values: [][]driver.Value{{
						row.id, row.login, row.hash, row.createdAt,
					}},
				}, nil
			}
		}
		return &fakeRows{columns: userColumns()}, nil
	case strings.Contains(query, "FROM records") && strings.Contains(query, "WHERE id"):
		id := fmt.Sprint(args[0].Value)
		row, ok := c.state.records[id]
		if !ok {
			return &fakeRows{columns: recordColumns()}, nil
		}
		return &fakeRows{
			columns: recordColumns(),
			values: [][]driver.Value{{
				row.id, row.ownerID, row.rtype, row.payload, row.meta, row.version, row.updatedAt, row.deleted,
			}},
		}, nil
	case strings.Contains(query, "FROM records") && strings.Contains(query, "WHERE owner_id"):
		owner := fmt.Sprint(args[0].Value)
		includeDeleted := !strings.Contains(query, "deleted = false")
		var typeFilter string
		if len(args) > 1 {
			typeFilter = fmt.Sprint(args[1].Value)
		}
		values := [][]driver.Value{}
		for _, row := range c.state.records {
			if row.ownerID != owner {
				continue
			}
			if !includeDeleted && row.deleted {
				continue
			}
			if typeFilter != "" && row.rtype != typeFilter {
				continue
			}
			values = append(values, []driver.Value{
				row.id, row.ownerID, row.rtype, row.payload, row.meta, row.version, row.updatedAt, row.deleted,
			})
		}
		return &fakeRows{columns: recordColumns(), values: values}, nil
	case strings.Contains(query, "FROM change_log"):
		owner := fmt.Sprint(args[0].Value)
		start := args[1].Value.(int64)
		limit := parseLimit(query)
		values := [][]driver.Value{}
		for _, row := range c.state.changes {
			if row.ownerID != owner || row.id <= start {
				continue
			}
			values = append(values, []driver.Value{
				row.id, row.recordID, row.ownerID, row.rtype, row.changeType, row.payload, row.meta, row.version, row.happenedAt,
			})
			if limit > 0 && len(values) >= limit {
				break
			}
		}
		return &fakeRows{columns: changeColumns(), values: values}, nil
	default:
		return &fakeRows{}, nil
	}
}

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

type fakeRows struct {
	columns []string
	values  [][]driver.Value
	idx     int
}

func (r *fakeRows) Columns() []string { return r.columns }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.idx])
	r.idx++
	return nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) { return int64(r), nil }

func parseLimit(query string) int {
	idx := strings.LastIndex(query, "LIMIT ")
	if idx == -1 {
		return 0
	}
	var limit int
	_, _ = fmt.Sscanf(query[idx:], "LIMIT %d", &limit)
	return limit
}

func userColumns() []string {
	return []string{"id", "login", "password_hash", "created_at"}
}

func recordColumns() []string {
	return []string{"id", "owner_id", "type", "payload", "meta", "version", "updated_at", "deleted"}
}

func changeColumns() []string {
	return []string{"id", "record_id", "owner_id", "type", "change", "payload", "meta", "version", "happened_at"}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	driverOnce.Do(func() {
		sql.Register("pgtest", fakeDriver{})
	})
	testState = &fakeState{}
	db, err := sql.Open("pgtest", "")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestNewDB(t *testing.T) {
	db, err := NewDB(config.Config{StorageBackend: "memory"})
	if err != nil || db != nil {
		t.Fatalf("expected nil db")
	}
	if _, err := NewDB(config.Config{StorageBackend: "postgres"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUserRepository(t *testing.T) {
	db := openTestDB(t)
	repo := NewUserRepository(db)
	user := domain.User{ID: "u1", Login: "login", CreatedAt: time.Now()}

	if _, err := repo.Create(context.Background(), user, "hash"); err != nil {
		t.Fatalf("create error: %v", err)
	}
	if _, err := repo.Create(context.Background(), user, "hash"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict")
	}
	if _, _, err := repo.FindByLogin(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found")
	}
	if err := repo.UpdatePasswordHash(context.Background(), "missing", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found")
	}
}

func TestUserRepositoryFindByID(t *testing.T) {
	db := openTestDB(t)
	repo := NewUserRepository(db)
	user := domain.User{ID: "u2", Login: "login2", CreatedAt: time.Now()}

	if _, err := repo.Create(context.Background(), user, "hash"); err != nil {
		t.Fatalf("create error: %v", err)
	}
	got, hash, err := repo.FindByID(context.Background(), "u2")
	if err != nil || got.Login != "login2" || hash != "hash" {
		t.Fatalf("find by id error")
	}
	if err := repo.UpdatePasswordHash(context.Background(), "u2", "hash2"); err != nil {
		t.Fatalf("update error: %v", err)
	}
}

func TestRecordRepository(t *testing.T) {
	db := openTestDB(t)
	repo := NewRecordRepository(db)
	now := time.Now().UTC()
	record := domain.Record{
		ID:        "r1",
		OwnerID:   "u1",
		Type:      domain.RecordTypeText,
		Payload:   domain.TextPayload{Text: "hello"},
		Meta:      domain.Metadata{Tags: []string{"tag"}},
		Version:   1,
		UpdatedAt: now,
	}
	if _, err := repo.Upsert(context.Background(), record); err != nil {
		t.Fatalf("upsert error: %v", err)
	}
	if _, err := repo.Get(context.Background(), "r1"); err != nil {
		t.Fatalf("get error: %v", err)
	}
	iter, err := repo.List(context.Background(), "u1", outbound.RecordFilter{Type: domain.RecordTypeText, Tag: "tag"})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	list, err := outbound.Collect(context.Background(), iter)
	if err != nil || len(list) != 1 {
		t.Fatalf("list error")
	}
	if err := repo.Delete(context.Background(), "r1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if err := repo.Delete(context.Background(), "r1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found")
	}
}

func TestRecordRepositoryFilters(t *testing.T) {
	db := openTestDB(t)
	repo := NewRecordRepository(db)
	now := time.Now().UTC()
	record := domain.Record{
		ID:      "r2",
		OwnerID: "u2",
		Type:    domain.RecordTypeText,
		Payload: domain.TextPayload{Text: "hello"},
		Meta: domain.Metadata{
			Title:       "Title",
			Description: "Desc",
			Tags:        []string{"tag1"},
			Attributes:  map[string]string{"key": "value"},
		},
		Version:   1,
		UpdatedAt: now,
	}
	if _, err := repo.Upsert(context.Background(), record); err != nil {
		t.Fatalf("upsert error: %v", err)
	}
	if err := repo.Delete(context.Background(), "r2"); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	iter, err := repo.List(context.Background(), "u2", outbound.RecordFilter{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("expected include deleted")
	}
	list, err := outbound.Collect(context.Background(), iter)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected include deleted")
	}
	iter, err = repo.List(context.Background(), "u2", outbound.RecordFilter{Tag: "tag1"})
	if err != nil {
		t.Fatalf("expected filtered out deleted")
	}
	list, err = outbound.Collect(context.Background(), iter)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected filtered out deleted")
	}
	iter, err = repo.List(context.Background(), "u2", outbound.RecordFilter{Query: "title"})
	if err != nil {
		t.Fatalf("expected filtered out deleted by query")
	}
	list, err = outbound.Collect(context.Background(), iter)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected filtered out deleted by query")
	}
	iter, err = repo.List(context.Background(), "u2", outbound.RecordFilter{IncludeDeleted: true, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("expected offset slice")
	}
	list, err = outbound.Collect(context.Background(), iter)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected offset slice")
	}
}

func TestChangeLogRepository(t *testing.T) {
	db := openTestDB(t)
	repo := NewChangeLogRepository(db)
	change := domain.RecordChange{
		RecordID:   "r1",
		OwnerID:    "u1",
		Type:       domain.RecordTypeText,
		Change:     domain.ChangeUpsert,
		Payload:    domain.TextPayload{Text: "hi"},
		Meta:       domain.Metadata{},
		Version:    1,
		HappenedAt: time.Now().UTC(),
	}
	if err := repo.Append(context.Background(), change); err != nil {
		t.Fatalf("append error: %v", err)
	}
	changes, cursor, err := repo.List(context.Background(), "u1", "0", 10)
	if err != nil || len(changes) != 1 || cursor == "" {
		t.Fatalf("list error")
	}
}

func TestRecordHelpers(t *testing.T) {
	if itoa(7) != "7" {
		t.Fatalf("itoa mismatch")
	}
	if hasTag([]string{"a", "b"}, "b") != true {
		t.Fatalf("hasTag mismatch")
	}
	record := domain.Record{
		Meta: domain.Metadata{
			Title:       "Hello",
			Description: "World",
			Tags:        []string{"tag"},
			Attributes:  map[string]string{"key": "value"},
		},
	}
	if !matchesQuery(record, "hello") {
		t.Fatalf("expected title match")
	}
	if !matchesQuery(record, "value") {
		t.Fatalf("expected attr match")
	}
	if matchesQuery(record, "missing") {
		t.Fatalf("expected no match")
	}
}
