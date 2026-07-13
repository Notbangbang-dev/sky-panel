package httpapi

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// Modrinth integration. Search/version listing is proxied through the panel
// (rather than called from the browser) so we present one origin, add a
// User-Agent Modrinth asks for, and aren't at the mercy of the browser's CORS
// handling of rate-limit responses. Install downloads a chosen file
// server-side and writes it into the server's mods/ or plugins/ folder via the
// existing daemon file-write path.

const (
	modrinthAPI     = "https://api.modrinth.com/v2"
	modrinthAPIHost = "api.modrinth.com"
	// Modrinth serves version files from this CDN host. Restricting installs to
	// it is the SSRF guard: the panel will only fetch from Modrinth, never an
	// attacker-supplied internal URL.
	modrinthCDNHost = "cdn.modrinth.com"
	// Matches the daemon's own decoded-file cap; a larger mod/pack can't be
	// installed this way and the user is told to use a modpack egg instead.
	maxModrinthFileBytes = 10 << 20
	// Cap proxied JSON so a hostile/huge upstream response can't blow memory.
	maxModrinthJSONBytes = 4 << 20
)

// The host allowlist on the install URL only covers the first hop, so we also
// re-validate every redirect target: without this a redirect from a Modrinth
// host to an internal address would be followed (SSRF). Only Modrinth hosts
// over https are ever followed.
var modrinthClient = &http.Client{
	Timeout: 25 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if req.URL.Scheme != "https" || (req.URL.Host != modrinthCDNHost && req.URL.Host != modrinthAPIHost) {
			return fmt.Errorf("refusing redirect to non-modrinth host %q", req.URL.Host)
		}
		return nil
	},
}

func modrinthUserAgent() string {
	return "sky-panel/0.18 (+https://github.com/Notbangbang-dev/sky-panel)"
}

// proxyModrinthGET fetches upstreamPath (relative to the Modrinth API base),
// size-caps the response, and relays it to the client as JSON.
func (d Deps) proxyModrinthGET(w http.ResponseWriter, upstreamURL string) {
	req, err := http.NewRequest(http.MethodGet, upstreamURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to build request")
		return
	}
	req.Header.Set("User-Agent", modrinthUserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := modrinthClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "modrinth_unreachable", "could not reach Modrinth")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModrinthJSONBytes))
	if err != nil {
		writeError(w, http.StatusBadGateway, "modrinth_error", "failed reading Modrinth response")
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "modrinth_error", fmt.Sprintf("Modrinth returned %d", resp.StatusCode))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// ModrinthSearch proxies GET /v2/search, translating our simple query params
// into Modrinth's facet syntax.
func (d Deps) ModrinthSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	projectType := q.Get("type") // mod | plugin | modpack | datapack | shader
	loader := q.Get("loader")    // fabric | forge | quilt | paper | spigot ...
	version := q.Get("version")  // game version, e.g. 1.21
	limit := q.Get("limit")
	offset := q.Get("offset")

	// facets is a JSON array of arrays: outer AND, inner OR.
	var facets [][]string
	if projectType != "" {
		facets = append(facets, []string{"project_type:" + projectType})
	}
	if loader != "" {
		facets = append(facets, []string{"categories:" + loader})
	}
	if version != "" {
		facets = append(facets, []string{"versions:" + version})
	}

	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	params.Set("index", "relevance")
	if limit == "" {
		limit = "20"
	}
	params.Set("limit", limit)
	if offset != "" {
		params.Set("offset", offset)
	}
	if len(facets) > 0 {
		params.Set("facets", buildFacets(facets))
	}

	d.proxyModrinthGET(w, modrinthAPI+"/search?"+params.Encode())
}

// buildFacets renders [["a:b"],["c:d"]] as Modrinth expects.
func buildFacets(facets [][]string) string {
	var groups []string
	for _, g := range facets {
		var quoted []string
		for _, f := range g {
			quoted = append(quoted, `"`+f+`"`)
		}
		groups = append(groups, "["+strings.Join(quoted, ",")+"]")
	}
	return "[" + strings.Join(groups, ",") + "]"
}

// ModrinthVersions proxies GET /v2/project/{id}/version filtered by loader and
// game version.
func (d Deps) ModrinthVersions(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "project is required")
		return
	}
	// project is a slug/id; keep it path-safe.
	if strings.ContainsAny(project, "/?#") {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid project id")
		return
	}

	params := url.Values{}
	if loader := r.URL.Query().Get("loader"); loader != "" {
		params.Set("loaders", `["`+loader+`"]`)
	}
	if version := r.URL.Query().Get("version"); version != "" {
		params.Set("game_versions", `["`+version+`"]`)
	}

	upstream := modrinthAPI + "/project/" + url.PathEscape(project) + "/version"
	if enc := params.Encode(); enc != "" {
		upstream += "?" + enc
	}
	d.proxyModrinthGET(w, upstream)
}

type modrinthInstallRequest struct {
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
	Folder      string `json:"folder"` // "mods" or "plugins"
}

// ModrinthInstall downloads a chosen Modrinth version file server-side and
// writes it into the server's mods/ or plugins/ directory.
func (d Deps) ModrinthInstall(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerForWrite(w, r, models.PermFiles)
	if server == nil {
		return
	}

	var req modrinthInstallRequest
	if err := decodeJSON(r, &req); err != nil || req.DownloadURL == "" || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "download_url and filename are required")
		return
	}

	folder := req.Folder
	if folder != "mods" && folder != "plugins" {
		writeError(w, http.StatusBadRequest, "bad_request", "folder must be mods or plugins")
		return
	}

	// Sanitise the filename to a single safe path segment ending in .jar.
	name := req.Filename
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") || !strings.HasSuffix(name, ".jar") {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid filename")
		return
	}

	// SSRF guard: only ever fetch from the Modrinth CDN.
	u, err := url.Parse(req.DownloadURL)
	if err != nil || u.Scheme != "https" || u.Host != modrinthCDNHost {
		writeError(w, http.StatusBadRequest, "bad_request", "download_url must be a modrinth CDN https url")
		return
	}

	dlReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to build download request")
		return
	}
	dlReq.Header.Set("User-Agent", modrinthUserAgent())

	resp, err := modrinthClient.Do(dlReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "modrinth_unreachable", "could not download from Modrinth")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, "modrinth_error", fmt.Sprintf("Modrinth download returned %d", resp.StatusCode))
		return
	}

	// Read one byte past the cap so we can distinguish "exactly at cap" from
	// "too big" and give a clear message.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxModrinthFileBytes+1))
	if err != nil {
		writeError(w, http.StatusBadGateway, "modrinth_error", "failed reading download")
		return
	}
	if len(data) > maxModrinthFileBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large",
			"that file is larger than 10MB — install it with a Modrinth modpack egg instead")
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	_, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{
		Action:        agenthub.ActionWriteFile,
		Path:          folder + "/" + name,
		ContentBase64: encoded,
	})
	if !ok {
		return
	}
	d.audit(r, "server.modrinth.install", server.ID, folder+"/"+name)
	writeJSON(w, http.StatusOK, map[string]string{"path": folder + "/" + name})
}
