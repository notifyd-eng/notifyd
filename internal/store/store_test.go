package store

import (
	"testing"

	"github.com/notifyd-eng/notifyd/internal/config"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()
	s, err := Open(config.StoreConfig{
		Driver: "sqlite3",
		DSN:    ":memory:",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInsertAndGet(t *testing.T) {
	s := setupTestDB(t)

	n := &Notification{
		ID:        "ntf_test1",
		Channel:   "webhook",
		Recipient: "https://example.com/hook",
		Subject:   "Test",
		Body:      "Hello world",
		Priority:  1,
	}

	if err := s.Insert(n); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := s.Get("ntf_test1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected notification, got nil")
	}
	if got.Channel != "webhook" {
		t.Errorf("channel = %q, want %q", got.Channel, "webhook")
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want %q", got.Status, "pending")
	}
}

func TestGetNotFound(t *testing.T) {
	s := setupTestDB(t)

	got, err := s.Get("ntf_nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestMarkSent(t *testing.T) {
	s := setupTestDB(t)

	n := &Notification{
		ID:        "ntf_sent1",
		Channel:   "email",
		Recipient: "user@example.com",
		Body:      "Test body",
	}
	s.Insert(n)

	if err := s.MarkSent("ntf_sent1"); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	got, _ := s.Get("ntf_sent1")
	if got.Status != "sent" {
		t.Errorf("status = %q, want %q", got.Status, "sent")
	}
	if got.SentAt == nil {
		t.Error("sent_at should not be nil")
	}
}

func TestListWithFilters(t *testing.T) {
	s := setupTestDB(t)

	for i, ch := range []string{"email", "webhook", "email", "slack"} {
		s.Insert(&Notification{
			ID:      "ntf_list" + string(rune('a'+i)),
			Channel: ch,
			Body:    "test",
		})
	}

	results, err := s.List(ListFilter{Channel: "email"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestStats(t *testing.T) {
	s := setupTestDB(t)

	s.Insert(&Notification{ID: "ntf_s1", Channel: "email", Body: "t"})
	s.Insert(&Notification{ID: "ntf_s2", Channel: "email", Body: "t"})
	s.MarkSent("ntf_s1")

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats["pending"] != 1 {
		t.Errorf("pending = %d, want 1", stats["pending"])
	}
	if stats["sent"] != 1 {
		t.Errorf("sent = %d, want 1", stats["sent"])
	}
}

func TestPendingBatch(t *testing.T) {
	s := setupTestDB(t)

	for i := 0; i < 5; i++ {
		s.Insert(&Notification{
			ID:      "ntf_batch" + string(rune('0'+i)),
			Channel: "webhook",
			Body:    "batch test",
		})
	}
	s.MarkSent("ntf_batch0")

	batch, err := s.PendingBatch(10)
	if err != nil {
		t.Fatalf("pending batch: %v", err)
	}
	if len(batch) != 4 {
		t.Errorf("got %d pending, want 4", len(batch))
	}
}
