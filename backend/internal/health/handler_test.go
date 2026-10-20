package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct {
	err error
}

func (f fakeDB) Ping(context.Context) error {
	return f.err
}

func TestHandlerDatabaseStatus(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   Response
	}{
		{
			name:       "database up",
			wantStatus: http.StatusOK,
			wantBody:   Response{Status: "ok", Database: true},
		},
		{
			name:       "database down",
			pingErr:    errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   Response{Status: "unavailable", Database: false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(fakeDB{err: test.pingErr})
			request := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
			recorder := httptest.NewRecorder()

			handler(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status %d, want %d", recorder.Code, test.wantStatus)
			}

			var body Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body != test.wantBody {
				t.Fatalf("body %+v, want %+v", body, test.wantBody)
			}
		})
	}
}
