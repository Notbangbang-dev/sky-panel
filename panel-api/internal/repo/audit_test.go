package repo

import "testing"

func TestAuditRecordAndList(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	audit := NewAudit(db)

	actor := newTestUser()
	if err := users.Create(actor); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := audit.Record(actor.ID, "node.create", "node-1", "my-node"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := audit.Record(actor.ID, "egg.delete", "egg-1", ""); err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := audit.List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Most recent first.
	if entries[0].Action != "egg.delete" || entries[1].Action != "node.create" {
		t.Errorf("unexpected order: %+v", entries)
	}
	if entries[1].Target != "node-1" || entries[1].Metadata != "my-node" {
		t.Errorf("unexpected entry fields: %+v", entries[1])
	}
}

func TestAuditListRespectsLimit(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	audit := NewAudit(db)

	actor := newTestUser()
	if err := users.Create(actor); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	for i := 0; i < 5; i++ {
		if err := audit.Record(actor.ID, "test.action", "", ""); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	entries, err := audit.List(2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected List(2) to return 2 entries, got %d", len(entries))
	}
}
