package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// loadServerForWrite is loadServerWithPermission plus a suspension gate: a
// suspended server accepts no state-changing operations from its owner or
// subusers (only admins may act on it). Read-only handlers keep using
// loadServerWithPermission so a suspended server's data stays viewable.
func (d Deps) loadServerForWrite(w http.ResponseWriter, r *http.Request, requiredPerm string) *models.Server {
	server := d.loadServerWithPermission(w, r, requiredPerm)
	if server == nil {
		return nil
	}
	if server.Suspended && !d.isAdmin(r) {
		writeError(w, http.StatusForbidden, "server_suspended", "this server is suspended by an administrator")
		return nil
	}
	return server
}

// validBackupFilename reports whether name is a safe backup archive filename:
// no path separators or traversal, no dotfile, and the expected .tar.zst
// extension the daemon produces. Mirrors the Modrinth install filename guard so
// a user-supplied filename can never escape the server's backup directory on
// restore/delete.
func validBackupFilename(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return false
	}
	return strings.HasSuffix(name, ".tar.zst")
}

// maxServerNameLen / maxVariables / maxVariableLen bound user-supplied server
// metadata so an oversized name or a flood of egg variables can't bloat the DB
// row or the container spec sent to the daemon.
const (
	maxServerNameLen = 100
	maxVariables     = 100
	maxVariableKey   = 128
	maxVariableLen   = 8192
)

// numericSettingMax bounds the admin-settable numeric settings so the raw
// key/value editor can't push the economy/quota knobs to absurd values (e.g.
// millions of coins per AFK heartbeat). A key here must parse as a non-negative
// integer no greater than its cap.
var numericSettingMax = map[string]int64{
	"afk.coins_per_heartbeat":     100_000,
	"afk.max_interval_seconds":    86_400,
	"afk.min_interval_seconds":    86_400,
	"daily_reward.amount":         10_000_000,
	"daily_reward.interval_hours": 8_760,
	"quota.default_cpu_percent":   100_000,
	"quota.default_databases":     10_000,
	"quota.default_memory_bytes":  1 << 50,
	"quota.default_disk_bytes":    1 << 50,
}

// booleanSettings must be exactly "true" or "false".
var booleanSettings = map[string]bool{
	"registration_enabled":      true,
	"maintenance.enabled":       true,
	"quota.allow_unlimited_cpu": true,
}

// validateSetting checks an admin-set setting value against a small schema for
// the known economy/quota/toggle keys, so out-of-range or malformed values are
// rejected with a clear 400 instead of silently corrupting runtime behaviour.
// Unknown keys are accepted (free-form), but every value is length-bounded.
func validateSetting(key, value string) (bool, string) {
	if len(value) > 100_000 {
		return false, "value is too large"
	}
	if max, ok := numericSettingMax[key]; ok {
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || n < 0 || n > max {
			return false, "value must be a non-negative integer within the allowed range for this setting"
		}
	}
	if booleanSettings[key] {
		if v := strings.TrimSpace(value); v != "true" && v != "false" {
			return false, "value must be 'true' or 'false'"
		}
	}
	return true, ""
}

// normalizeEmail lower-cases and trims an email so lookups and the uniqueness
// constraint treat case/whitespace variants as the same address.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// trimName trims surrounding whitespace and clamps a server/display name to a
// sane maximum length.
func trimName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > maxServerNameLen {
		name = name[:maxServerNameLen]
	}
	return name
}

// validVariables reports whether an egg-variable override map is within bounds.
func validVariables(vars map[string]string) bool {
	if len(vars) > maxVariables {
		return false
	}
	for k, v := range vars {
		if len(k) > maxVariableKey || len(v) > maxVariableLen {
			return false
		}
	}
	return true
}

// validRelPath is a defense-in-depth check for file-manager paths handled by
// the panel before dispatch to the daemon (which is the authoritative guard).
// It rejects absolute paths and parent-directory traversal so the API can
// return a clean 400 instead of relaying a daemon error, and so a future daemon
// containment bug is harder to reach. An empty path means "server root".
func validRelPath(p string) bool {
	if p == "" {
		return true
	}
	if len(p) > 1024 {
		return false
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		return false
	}
	// Reject any ".." path segment (covers a/../b, ../x, and a trailing ..).
	for _, seg := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return false
		}
	}
	// NUL and other control bytes have no place in a path.
	for _, r := range p {
		if r < 0x20 {
			return false
		}
	}
	return true
}
