package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"carjournal/internal/httpapi"
	"carjournal/internal/notes"
)

func TestNotesAPI(t *testing.T) {
	server := httpapi.New(newFakeStore())

	rec := request(t, server, http.MethodGet, "/api/notes", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, server, http.MethodPost, "/api/notes", `{"title":"  Масло  ","body":"5w30"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	created := decodeNote(t, rec)
	if created.Title != "Масло" || created.ID == 0 {
		t.Fatalf("created note: %+v", created)
	}

	rec = request(t, server, http.MethodPost, "/api/notes", `{"title":"   "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty title: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, server, http.MethodPut, "/api/notes/1", `{"title":"Масло","body":"5w40"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	updated := decodeNote(t, rec)
	if updated.Body != "5w40" {
		t.Fatalf("updated body: %+v", updated)
	}

	rec = request(t, server, http.MethodPut, "/api/notes/99", `{"title":"Нет","body":""}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing update: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, server, http.MethodDelete, "/api/notes/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, server, http.MethodDelete, "/api/notes/1", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second delete: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, server, http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("health: %d %s", rec.Code, rec.Body.String())
	}
}

func request(t *testing.T, server *httpapi.Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}

func decodeNote(t *testing.T, rec *httptest.ResponseRecorder) notes.Note {
	t.Helper()
	var note notes.Note
	if err := json.Unmarshal(rec.Body.Bytes(), &note); err != nil {
		t.Fatal(err)
	}
	return note
}

type fakeStore struct {
	mu    sync.Mutex
	next  int64
	items map[int64]notes.Note
}

func newFakeStore() *fakeStore {
	return &fakeStore{next: 1, items: map[int64]notes.Note{}}
}

func (f *fakeStore) List(context.Context) ([]notes.Note, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]notes.Note, 0, len(f.items))
	for _, note := range f.items {
		items = append(items, note)
	}
	return items, nil
}

func (f *fakeStore) Create(_ context.Context, title, body string) (notes.Note, error) {
	title, body, err := notes.Normalize(title, body)
	if err != nil {
		return notes.Note{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	note := notes.Note{ID: f.next, Title: title, Body: body, CreatedAt: now, UpdatedAt: now}
	f.next++
	f.items[note.ID] = note
	return note, nil
}

func (f *fakeStore) Update(_ context.Context, id int64, title, body string) (notes.Note, error) {
	title, body, err := notes.Normalize(title, body)
	if err != nil {
		return notes.Note{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	note, ok := f.items[id]
	if !ok {
		return notes.Note{}, notes.ErrNotFound
	}
	note.Title = title
	note.Body = body
	note.UpdatedAt = time.Now().UTC()
	f.items[id] = note
	return note, nil
}

var _ notes.Store = (*fakeStore)(nil)

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return notes.ErrNotFound
	}
	delete(f.items, id)
	return nil
}
