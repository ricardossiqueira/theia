package deviceplatform

import (
	"io"
	"net/http"
)

// deviceRuntime is the subset of *runtime.Runtime the HTTP server needs.
// Kept as an interface so tests can use a fake instead of a real provisioned
// core, and named to avoid colliding with the stdlib "runtime" package.
type deviceRuntime interface {
	DeviceInfoJSON() ([]byte, error)
	Pair(body []byte) ([]byte, error)
	Provision(body []byte) ([]byte, error)
}

// NewHandler exposes exactly the three endpoints iot-gateway's
// internal/devicev2 client calls (HTTPInspector.Inspect and
// SessionClient.Pair/Provision): GET /v1/device-info, POST /v1/pair and
// POST /v1/provision. /v1/provision-status from
// DEVICE_PLATFORM_V2_IMPLEMENTATION.md is intentionally not implemented: no
// caller in iot-gateway requests it, and DeviceInfo's own
// provisioning_state already carries that information.
func NewHandler(rt deviceRuntime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/device-info", func(w http.ResponseWriter, r *http.Request) {
		body, err := rt.DeviceInfoJSON()
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, body)
	})
	mux.HandleFunc("POST /v1/pair", func(w http.ResponseWriter, r *http.Request) {
		handlePost(w, r, rt.Pair)
	})
	mux.HandleFunc("POST /v1/provision", func(w http.ResponseWriter, r *http.Request) {
		handlePost(w, r, rt.Provision)
	})
	return mux
}

func handlePost(w http.ResponseWriter, r *http.Request, call func([]byte) ([]byte, error)) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		httpError(w, err)
		return
	}
	response, err := call(body)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, response)
}

func writeJSON(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func httpError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
