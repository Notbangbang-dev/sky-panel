package httpapi

import "net/http"

func (d Deps) AdminGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := d.Settings.All()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

type setSettingRequest struct {
	Value string `json:"value"`
}

func (d Deps) AdminSetSetting(w http.ResponseWriter, r *http.Request) {
	var req setSettingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	key := pathParam(r, "key")
	if err := d.Settings.Set(key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save setting")
		return
	}

	d.audit(r, "settings.set", key, req.Value)
	w.WriteHeader(http.StatusNoContent)
}
