package serversvc

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func TestResolveVariablesUsesDefaultsAndOverrides(t *testing.T) {
	vars := []models.EggVariable{
		{Name: "Server Jar", Env: "SERVER_JAR", Default: "server.jar", UserEditable: true},
		{Name: "Version", Env: "VERSION", Default: "1.20.1", UserEditable: false},
	}

	resolved := resolveVariables(vars, map[string]string{
		"SERVER_JAR": "paper.jar",
		"VERSION":    "9.9.9", // not user editable: must be ignored
	})

	if resolved["SERVER_JAR"] != "paper.jar" {
		t.Errorf("expected override to apply to editable var, got %q", resolved["SERVER_JAR"])
	}
	if resolved["VERSION"] != "1.20.1" {
		t.Errorf("expected non-editable var to keep its default, got %q", resolved["VERSION"])
	}
}

func TestSubstituteReplacesKnownTokensAndLeavesUnknown(t *testing.T) {
	values := map[string]string{"SERVER_PORT": "25565", "SERVER_MEMORY": "1024"}

	got := substitute(`java -Xmx{{SERVER_MEMORY}}M -jar server.jar --port {{SERVER_PORT}} {{UNKNOWN}}`, values)
	want := `java -Xmx1024M -jar server.jar --port 25565 {{UNKNOWN}}`

	if got != want {
		t.Errorf("substitute() = %q, want %q", got, want)
	}
}

func TestTokenizeCommandSplitsOnWhitespace(t *testing.T) {
	got := tokenizeCommand(`java -Xmx1024M -jar server.jar`)
	want := []string{"java", "-Xmx1024M", "-jar", "server.jar"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokenizeCommand() = %v, want %v", got, want)
	}
}

func TestTokenizeCommandRespectsQuotedSpans(t *testing.T) {
	got := tokenizeCommand(`nginx -g "daemon off;"`)
	want := []string{"nginx", "-g", "daemon off;"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokenizeCommand() = %v, want %v", got, want)
	}
}

func TestResolveVariablesEnvKeysAreEggDefinedEnvNames(t *testing.T) {
	vars := []models.EggVariable{
		{Name: "A", Env: "ENV_A", Default: "a", UserEditable: true},
		{Name: "B", Env: "ENV_B", Default: "b", UserEditable: true},
	}

	resolved := resolveVariables(vars, nil)

	keys := make([]string, 0, len(resolved))
	for k := range resolved {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	want := []string{"ENV_A", "ENV_B"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("resolveVariables() keys = %v, want %v", keys, want)
	}
}
