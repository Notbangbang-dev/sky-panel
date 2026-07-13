package httpapi

import "testing"

func TestValidBackupFilename(t *testing.T) {
	good := []string{"20260701-120000.tar.zst", "backup-1782000000.tar.zst"}
	for _, n := range good {
		if !validBackupFilename(n) {
			t.Errorf("expected %q to be valid", n)
		}
	}
	bad := []string{
		"", "backup.tar.gz", "backup.zip", "../secret.tar.zst",
		"a/b.tar.zst", "a\\b.tar.zst", ".hidden.tar.zst",
		"..tar.zst" + "/../x", "plain.txt",
	}
	for _, n := range bad {
		if validBackupFilename(n) {
			t.Errorf("expected %q to be rejected", n)
		}
	}
}

func TestValidRelPath(t *testing.T) {
	good := []string{"", "server.properties", "plugins/config.yml", "a/b/c.txt", "world/level.dat"}
	for _, p := range good {
		if !validRelPath(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}
	bad := []string{"/etc/passwd", "../../etc/passwd", "plugins/../../x", "a/../../b", "\\windows\\system32", "bad\x00byte"}
	for _, p := range bad {
		if validRelPath(p) {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}

func TestValidVariables(t *testing.T) {
	if !validVariables(nil) {
		t.Error("nil variables should be valid")
	}
	if !validVariables(map[string]string{"MEMORY": "1024M", "EULA": "true"}) {
		t.Error("small variable map should be valid")
	}
	big := make(map[string]string, maxVariables+1)
	for i := 0; i < maxVariables+1; i++ {
		big[string(rune('a'+i%26))+string(rune('0'+i/26))] = "x"
	}
	if validVariables(big) {
		t.Error("too many variables should be rejected")
	}
	huge := map[string]string{"X": string(make([]byte, maxVariableLen+1))}
	if validVariables(huge) {
		t.Error("oversized variable value should be rejected")
	}
}

func TestTrimName(t *testing.T) {
	if got := trimName("  hello  "); got != "hello" {
		t.Errorf("trimName whitespace: got %q", got)
	}
	long := make([]byte, maxServerNameLen+50)
	for i := range long {
		long[i] = 'a'
	}
	if got := trimName(string(long)); len(got) != maxServerNameLen {
		t.Errorf("trimName length: got %d, want %d", len(got), maxServerNameLen)
	}
}
