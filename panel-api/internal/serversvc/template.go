package serversvc

import (
	"strings"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// resolveVariables merges egg-defined defaults with caller-supplied
// overrides. Overrides are keyed by each variable's Env name (a stable
// machine identifier), not its human-readable Name. Only variables marked
// UserEditable may be overridden; anything else always uses its egg-defined
// default, regardless of what was passed in the request.
func resolveVariables(vars []models.EggVariable, overrides map[string]string) map[string]string {
	resolved := make(map[string]string, len(vars))
	for _, v := range vars {
		value := v.Default
		if v.UserEditable {
			if ov, ok := overrides[v.Env]; ok {
				value = ov
			}
		}
		resolved[v.Env] = value
	}
	return resolved
}

// substitute replaces every {{TOKEN}} occurrence in s using values. Unknown
// tokens are left as-is (visible in the console rather than silently
// producing an empty string, which would be far more confusing to debug).
func substitute(s string, values map[string]string) string {
	var b strings.Builder
	for {
		start := strings.Index(s, "{{")
		if start < 0 {
			b.WriteString(s)
			break
		}
		end := strings.Index(s[start:], "}}")
		if end < 0 {
			b.WriteString(s)
			break
		}
		end += start

		b.WriteString(s[:start])
		token := strings.TrimSpace(s[start+2 : end])
		if v, ok := values[token]; ok {
			b.WriteString(v)
		} else {
			b.WriteString(s[start : end+2])
		}
		s = s[end+2:]
	}
	return b.String()
}

// tokenizeCommand splits a startup command string into argv the same way
// sky-daemon's container Cmd expects: whitespace-separated, but a
// double-quoted span counts as one token (so `-g "daemon off;"` becomes
// ["-g", "daemon off;"]). The resulting tokens are exec'd directly with no
// shell involved downstream, so shell metacharacters are inert by
// construction — the same safety property cloud-panel's launcher relies on.
func tokenizeCommand(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	hasCur := false

	flush := func() {
		if hasCur {
			tokens = append(tokens, cur.String())
			cur.Reset()
			hasCur = false
		}
	}

	for _, r := range s {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			hasCur = true
		case r == ' ' || r == '\t':
			if inQuotes {
				cur.WriteRune(r)
			} else {
				flush()
			}
		default:
			cur.WriteRune(r)
			hasCur = true
		}
	}
	flush()

	return tokens
}
