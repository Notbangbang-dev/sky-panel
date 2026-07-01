package repo

import "testing"

func TestSettingsSetGetAndOverwrite(t *testing.T) {
	s := NewSettings(newTestDB(t))

	if _, found, err := s.Get("site_name"); err != nil {
		t.Fatalf("Get: %v", err)
	} else if found {
		t.Error("expected site_name to be unset initially")
	}

	if err := s.Set("site_name", "Sky Panel"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	value, found, err := s.Get("site_name")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || value != "Sky Panel" {
		t.Errorf("expected found=true value='Sky Panel', got found=%v value=%q", found, value)
	}

	if err := s.Set("site_name", "Renamed Panel"); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}
	value, _, _ = s.Get("site_name")
	if value != "Renamed Panel" {
		t.Errorf("expected overwritten value, got %q", value)
	}
}

func TestSettingsAll(t *testing.T) {
	s := NewSettings(newTestDB(t))

	s.Set("a", "1")
	s.Set("b", "2")

	all, err := s.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if all["a"] != "1" || all["b"] != "2" || len(all) != 2 {
		t.Errorf("unexpected settings map: %+v", all)
	}
}
