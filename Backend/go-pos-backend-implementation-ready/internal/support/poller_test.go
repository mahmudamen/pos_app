package support

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakeFetcher struct {
	result FetchResult
	err    error
	calls  int
}

func (f *fakeFetcher) Fetch(context.Context) (FetchResult, error) {
	f.calls++
	return f.result, f.err
}

type fakeStore struct {
	ingested []InboundMessage
	failAt   int
	saved    map[string]SyncCursor
}

func (s *fakeStore) Ingest(_ context.Context, m InboundMessage) (bool, error) {
	if s.failAt > 0 && len(s.ingested) == s.failAt-1 {
		return false, errors.New("boom")
	}
	s.ingested = append(s.ingested, m)
	return true, nil
}

func (s *fakeStore) Cursor(context.Context, string) (SyncCursor, error) { return SyncCursor{}, nil }

func (s *fakeStore) SaveCursor(_ context.Context, mailbox string, c SyncCursor) error {
	if s.saved == nil {
		s.saved = map[string]SyncCursor{}
	}
	s.saved[mailbox] = c
	return nil
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestPollerPollOnce_StoresAllAndAdvancesCursor(t *testing.T) {
	fetcher := &fakeFetcher{result: FetchResult{
		Mailbox: "INBOX",
		Messages: []InboundMessage{
			{Mailbox: "INBOX", ProviderRef: "1", Subject: "a"},
			{Mailbox: "INBOX", ProviderRef: "2", Subject: "b"},
		},
		Cursor: SyncCursor{UIDValidity: 7, LastUID: 2},
	}}
	store := &fakeStore{}
	p := NewPoller(fetcher, store, 0, testLogger())

	if err := p.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if len(store.ingested) != 2 {
		t.Fatalf("ingested %d messages, want 2", len(store.ingested))
	}
	if got := store.saved["INBOX"]; got != (SyncCursor{UIDValidity: 7, LastUID: 2}) {
		t.Fatalf("saved cursor = %+v", got)
	}
}

func TestPollerPollOnce_DoesNotAdvanceCursorOnStoreFailure(t *testing.T) {
	fetcher := &fakeFetcher{result: FetchResult{
		Mailbox: "INBOX",
		Messages: []InboundMessage{
			{Mailbox: "INBOX", ProviderRef: "1"},
			{Mailbox: "INBOX", ProviderRef: "2"},
		},
		Cursor: SyncCursor{UIDValidity: 7, LastUID: 2},
	}}
	store := &fakeStore{failAt: 2}
	p := NewPoller(fetcher, store, 0, testLogger())

	if err := p.PollOnce(context.Background()); err == nil {
		t.Fatal("expected an error when a message fails to store")
	}
	if _, ok := store.saved["INBOX"]; ok {
		t.Fatal("cursor must not advance after a partial ingest")
	}
}

func TestPollerPollOnce_FetchError(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("imap down")}
	p := NewPoller(fetcher, &fakeStore{}, 0, testLogger())
	if err := p.PollOnce(context.Background()); err == nil {
		t.Fatal("expected fetch error to propagate")
	}
}
