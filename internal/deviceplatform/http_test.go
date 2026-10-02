package deviceplatform

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRuntime struct {
	deviceInfo        []byte
	deviceInfoErr     error
	lastPairBody      []byte
	pairResponse      []byte
	pairErr           error
	lastProvisionBody []byte
	provisionResponse []byte
	provisionErr      error
}

func (f *fakeRuntime) DeviceInfoJSON() ([]byte, error) { return f.deviceInfo, f.deviceInfoErr }
func (f *fakeRuntime) Pair(body []byte) ([]byte, error) {
	f.lastPairBody = append([]byte(nil), body...)
	return f.pairResponse, f.pairErr
}
func (f *fakeRuntime) Provision(body []byte) ([]byte, error) {
	f.lastProvisionBody = append([]byte(nil), body...)
	return f.provisionResponse, f.provisionErr
}

func TestHandlerDeviceInfo(t *testing.T) {
	rt := &fakeRuntime{deviceInfo: []byte(`{"protocol":"iot-device-v1"}`)}
	handler := NewHandler(rt)
	req := httptest.NewRequest(http.MethodGet, "/v1/device-info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"protocol":"iot-device-v1"}` {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestHandlerDeviceInfoError(t *testing.T) {
	rt := &fakeRuntime{deviceInfoErr: errors.New("boom")}
	handler := NewHandler(rt)
	req := httptest.NewRequest(http.MethodGet, "/v1/device-info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerPairForwardsBodyAndResponse(t *testing.T) {
	rt := &fakeRuntime{pairResponse: []byte(`{"session_id":"s"}`)}
	handler := NewHandler(rt)
	req := httptest.NewRequest(http.MethodPost, "/v1/pair", strings.NewReader(`{"version":1}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"session_id":"s"}` {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if string(rt.lastPairBody) != `{"version":1}` {
		t.Fatalf("Pair received %q", rt.lastPairBody)
	}
}

func TestHandlerProvisionError(t *testing.T) {
	rt := &fakeRuntime{provisionErr: errors.New("session expired")}
	handler := NewHandler(rt)
	req := httptest.NewRequest(http.MethodPost, "/v1/provision", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerRejectsUnknownRoute(t *testing.T) {
	handler := NewHandler(&fakeRuntime{})
	req := httptest.NewRequest(http.MethodGet, "/v1/provision-status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (no provision-status endpoint)", rec.Code)
	}
}
