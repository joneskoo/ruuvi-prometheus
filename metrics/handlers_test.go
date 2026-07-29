package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRoot(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"root serves description", "/", http.StatusOK, "ruuvi-prometheus"},
		{"unknown path 404s", "/nope", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handleRoot(rr, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestMetricsEndpoint verifies the mux wires /metrics to the Prometheus handler
// and that observed Ruuvi readings are exported there.
func TestMetricsEndpoint(t *testing.T) {
	clearDevice(t)
	ObserveRuuvi(reading(t, testAddr, v5Frame))

	rr := httptest.NewRecorder()
	Handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "ruuvi_frames_total") {
		t.Errorf("/metrics body missing ruuvi_frames_total; got:\n%s", rr.Body.String())
	}
}
