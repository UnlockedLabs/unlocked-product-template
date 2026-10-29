package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionRoute(t *testing.T) {
	v := Version{Version: "1.2.3", Commit: "abc1234", Branch: "main"}
	tests := []struct {
		name       string
		prefix     string
		method     string
		path       string
		wantStatus int
	}{
		{name: "no prefix", prefix: "", method: http.MethodGet, path: "/api/version", wantStatus: http.StatusOK},
		{name: "prefix", prefix: "/example", method: http.MethodGet, path: "/example/api/version", wantStatus: http.StatusOK},
		{name: "prefix with trailing slash", prefix: "/example/", method: http.MethodGet, path: "/example/api/version", wantStatus: http.StatusOK},
		{name: "outside the prefix", prefix: "/example", method: http.MethodGet, path: "/api/version", wantStatus: http.StatusNotFound},
		{name: "wrong method", prefix: "/example", method: http.MethodPost, path: "/example/api/version", wantStatus: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHandler(tt.prefix, v).ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got Version
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got != v {
				t.Fatalf("body = %+v, want %+v", got, v)
			}
		})
	}
}

func TestHealthcheckRoute(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "under the prefix", path: "/example/api/healthcheck", wantStatus: http.StatusOK},
		{name: "outside the prefix", path: "/api/healthcheck", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHandler("/example", Version{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && w.Body.String() != "OK" {
				t.Fatalf("body = %q, want %q", w.Body.String(), "OK")
			}
		})
	}
}

func TestVersionFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Version
	}{
		{
			name: "set",
			env:  map[string]string{"APP_VERSION": "1.2.3", "GIT_COMMIT": "abc1234", "GIT_BRANCH": "main"},
			want: Version{Version: "1.2.3", Commit: "abc1234", Branch: "main"},
		},
		{
			name: "unset",
			env:  map[string]string{"APP_VERSION": "", "GIT_COMMIT": "", "GIT_BRANCH": ""},
			want: Version{Version: "unknown", Commit: "unknown", Branch: "unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, val := range tt.env {
				t.Setenv(k, val)
			}
			if got := VersionFromEnv(); got != tt.want {
				t.Fatalf("VersionFromEnv() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
