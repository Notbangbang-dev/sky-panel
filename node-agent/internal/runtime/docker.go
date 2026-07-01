package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Docker talks directly to the Docker Engine API over a unix socket using
// only the standard library — no moby/docker SDK dependency. This keeps
// node-agent's dependency footprint tiny and gives full control over the one
// part of the API (attach/hijack) that needs a raw connection anyway.
type Docker struct {
	apiVersion string
	socketPath string
	client     *http.Client
}

// NewDocker returns a Docker runtime client talking to the Engine API over
// the given unix socket path (typically "/var/run/docker.sock").
func NewDocker(socketPath string) *Docker {
	return &Docker{
		apiVersion: "v1.43",
		socketPath: socketPath,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
			Timeout: 30 * time.Second,
		},
	}
}

func (d *Docker) url(path string) string {
	return "http://docker/" + d.apiVersion + path
}

func (d *Docker) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, d.url(path), reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker api %s %s: %w", method, path, err)
	}
	return resp, nil
}

// doExpect issues the request and returns an error (including the daemon's
// JSON error message body, if present) unless the status matches one of ok.
func (d *Docker) doExpect(ctx context.Context, method, path string, body any, ok ...int) (*http.Response, error) {
	resp, err := d.do(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	for _, want := range ok {
		if resp.StatusCode == want {
			return resp, nil
		}
	}
	defer resp.Body.Close()
	return nil, fmt.Errorf("docker api %s %s: unexpected status %s: %s", method, path, resp.Status, readDockerError(resp.Body))
}

func readDockerError(r io.Reader) string {
	var e struct {
		Message string `json:"message"`
	}
	b, _ := io.ReadAll(r)
	if json.Unmarshal(b, &e) == nil && e.Message != "" {
		return e.Message
	}
	return string(b)
}

type createContainerRequest struct {
	Image      string            `json:"Image"`
	Cmd        []string          `json:"Cmd,omitempty"`
	Env        []string          `json:"Env,omitempty"`
	WorkingDir string            `json:"WorkingDir,omitempty"`
	Labels     map[string]string `json:"Labels,omitempty"`
	Tty        bool              `json:"Tty"`
	OpenStdin  bool              `json:"OpenStdin"`
	HostConfig hostConfig        `json:"HostConfig"`
}

type hostConfig struct {
	Binds        []string                 `json:"Binds,omitempty"`
	PortBindings map[string][]portBinding `json:"PortBindings,omitempty"`
	Memory       int64                    `json:"Memory,omitempty"`
	NanoCPUs     int64                    `json:"NanoCpus,omitempty"`
	CapDrop      []string                 `json:"CapDrop,omitempty"`
	SecurityOpt  []string                 `json:"SecurityOpt,omitempty"`
}

type portBinding struct {
	HostPort string `json:"HostPort"`
}

func toCreateRequest(spec ContainerSpec) createContainerRequest {
	bindings := make(map[string][]portBinding, len(spec.PortBindings))
	for containerPort, hostPort := range spec.PortBindings {
		bindings[containerPort] = []portBinding{{HostPort: hostPort}}
	}

	return createContainerRequest{
		Image:      spec.Image,
		Cmd:        spec.Cmd,
		Env:        spec.Env,
		WorkingDir: spec.WorkingDir,
		Labels:     spec.Labels,
		Tty:        false,
		OpenStdin:  true,
		HostConfig: hostConfig{
			Binds:        spec.Binds,
			PortBindings: bindings,
			Memory:       spec.MemoryBytes,
			NanoCPUs:     spec.NanoCPUs,
			// Secure-by-default: no capabilities beyond what the image
			// itself needs, no privilege escalation.
			CapDrop:     []string{"ALL"},
			SecurityOpt: []string{"no-new-privileges"},
		},
	}
}

func (d *Docker) Create(ctx context.Context, spec ContainerSpec) (string, error) {
	path := "/containers/create"
	if spec.Name != "" {
		path += "?name=" + spec.Name
	}

	resp, err := d.doExpect(ctx, http.MethodPost, path, toCreateRequest(spec), http.StatusCreated)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}
	return out.ID, nil
}

func (d *Docker) Start(ctx context.Context, id string) error {
	resp, err := d.doExpect(ctx, http.MethodPost, "/containers/"+id+"/start", nil, http.StatusNoContent, http.StatusNotModified)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (d *Docker) Stop(ctx context.Context, id string, timeout time.Duration) error {
	seconds := int(timeout.Seconds())
	resp, err := d.doExpect(ctx, http.MethodPost, fmt.Sprintf("/containers/%s/stop?t=%d", id, seconds), nil,
		http.StatusNoContent, http.StatusNotModified)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (d *Docker) Kill(ctx context.Context, id string) error {
	resp, err := d.doExpect(ctx, http.MethodPost, "/containers/"+id+"/kill", nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (d *Docker) Remove(ctx context.Context, id string) error {
	resp, err := d.doExpect(ctx, http.MethodDelete, "/containers/"+id+"?force=true", nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (d *Docker) Inspect(ctx context.Context, id string) (ContainerState, error) {
	resp, err := d.doExpect(ctx, http.MethodGet, "/containers/"+id+"/json", nil, http.StatusOK)
	if err != nil {
		return ContainerState{}, err
	}
	defer resp.Body.Close()

	var out struct {
		ID    string `json:"Id"`
		State struct {
			Running  bool `json:"Running"`
			ExitCode int  `json:"ExitCode"`
		} `json:"State"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ContainerState{}, fmt.Errorf("decode inspect response: %w", err)
	}

	return ContainerState{ID: out.ID, Running: out.State.Running, ExitCode: out.State.ExitCode}, nil
}

func (d *Docker) Stats(ctx context.Context, id string) (Stats, error) {
	resp, err := d.doExpect(ctx, http.MethodGet, "/containers/"+id+"/stats?stream=false", nil, http.StatusOK)
	if err != nil {
		return Stats{}, err
	}
	defer resp.Body.Close()

	var raw dockerStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Stats{}, fmt.Errorf("decode stats response: %w", err)
	}

	return parseDockerStats(raw), nil
}
