package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// Instance appearance is admin-controlled and read by every client (including
// the login page, before auth), so it lives in the public settings surface.
// Values are plain settings keys written through the existing admin settings
// endpoint; this handler just bundles the three appearance keys into one
// unauthenticated read.

const (
	settingAppearanceTheme      = "appearance.theme_preset"
	settingAppearanceCustomCSS  = "appearance.custom_css"
	settingAppearanceBackground = "appearance.background"
	settingMaintenanceEnabled   = "maintenance.enabled"
	settingMaintenanceMessage   = "maintenance.message"
)

type appearanceResponse struct {
	ThemePreset string `json:"theme_preset"`
	CustomCSS   string `json:"custom_css"`
	Background  string `json:"background"`
}

func (d Deps) getSetting(key string) string {
	v, found, err := d.Settings.Get(key)
	if err != nil || !found {
		return ""
	}
	return v
}

// PublicAppearance returns the instance-wide look (theme preset id, custom CSS,
// and a JSON background config), applied app-wide by the frontend.
func (d Deps) PublicAppearance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, appearanceResponse{
		ThemePreset: d.getSetting(settingAppearanceTheme),
		CustomCSS:   d.getSetting(settingAppearanceCustomCSS),
		Background:  d.getSetting(settingAppearanceBackground),
	})
}

type maintenanceResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

func (d Deps) maintenanceEnabled() bool {
	return d.getSetting(settingMaintenanceEnabled) == "true"
}

// MaintenanceStatus is public so the frontend can show a maintenance screen to
// signed-out visitors too.
func (d Deps) MaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, maintenanceResponse{
		Enabled: d.maintenanceEnabled(),
		Message: d.getSetting(settingMaintenanceMessage),
	})
}

// maintenanceGate blocks authenticated non-admin API calls while maintenance
// mode is on, so an admin can freeze the panel for everyone else (and still
// use it themselves to turn it back off). It runs after RequireAuth, so the
// role is available; login/refresh and public endpoints are outside this group
// and stay reachable.
func (d Deps) maintenanceGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.maintenanceEnabled() {
			claims, ok := auth.FromContext(r.Context())
			if !ok || claims.Role != string(models.RoleAdmin) {
				writeError(w, http.StatusServiceUnavailable, "maintenance",
					"the panel is under maintenance — try again shortly")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
