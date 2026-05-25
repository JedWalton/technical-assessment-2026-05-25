package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/httpapi"
	"github.com/JedWalton/technical-assessment-2026-05-25/internal/order"
)

func TestPOSTOrders_validOrderAccepted(t *testing.T) {
	t.Parallel()

	adder := &fakeAdder{}
	h := httpapi.New(adder, testLogger())

	body := `{"customer_number":"C1","address":"1 High St","postcode":"AB12 3CD","placed_at":"2026-05-25T10:00:00Z"}`
	rec := postOrders(t, h, body, "application/json")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if adder.addCount() != 1 {
		t.Fatalf("addCount = %d, want 1", adder.addCount())
	}
	if adder.last().CustomerNumber != "C1" {
		t.Fatalf("order = %+v", adder.last())
	}
	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("missing X-Request-ID response header")
	}
}

func TestPOSTOrders_validationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing customer number",
			body:       `{"address":"1 High St","postcode":"AB1"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "CUSTOMER_NUMBER_REQUIRED",
		},
		{
			name:       "missing address",
			body:       `{"customer_number":"C1","postcode":"AB1"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "ADDRESS_REQUIRED",
		},
		{
			name:       "empty postcode",
			body:       `{"customer_number":"C1","address":"1 High St","postcode":""}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "POSTCODE_EMPTY",
		},
		{
			name:       "postcode too long",
			body:       `{"customer_number":"C1","address":"1 High St","postcode":"123456789"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "POSTCODE_TOO_LONG",
		},
		{
			name:       "postcode invalid char",
			body:       `{"customer_number":"C1","address":"1 High St","postcode":"AB1-2"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "POSTCODE_INVALID_CHAR",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := httpapi.New(&fakeAdder{}, testLogger())
			rec := postOrders(t, h, tt.body, "application/json")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var resp errorBody
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", resp.Code, tt.wantCode)
			}
			if resp.Error == "" {
				t.Fatal("error message is empty")
			}
		})
	}
}

func TestPOSTOrders_malformedJSON(t *testing.T) {
	t.Parallel()

	rec := postOrders(t, httpapi.New(&fakeAdder{}, testLogger()), `{`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPOSTOrders_unknownFieldsRejected(t *testing.T) {
	t.Parallel()

	body := `{"customer_number":"C1","address":"1 High St","postcode":"AB1","extra":true}`
	rec := postOrders(t, httpapi.New(&fakeAdder{}, testLogger()), body, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPOSTOrders_oversizedBody(t *testing.T) {
	t.Parallel()

	// Valid JSON shape so the limit is hit while decoding, not on syntax error.
	body := `{"customer_number":"C1","address":"` +
		strings.Repeat("x", httpapi.MaxRequestBodyBytes) +
		`","postcode":"AB1"}`
	rec := postOrders(t, httpapi.New(&fakeAdder{}, testLogger()), body, "application/json")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestPOSTOrders_wrongMethod(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rec := httptest.NewRecorder()
	httpapi.New(&fakeAdder{}, testLogger()).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestPOSTOrders_unsupportedContentType(t *testing.T) {
	t.Parallel()

	rec := postOrders(t, httpapi.New(&fakeAdder{}, testLogger()),
		`{"customer_number":"C1","address":"A","postcode":"AB1"}`, "text/plain")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}

func TestPOSTOrders_adderErrorReturns500(t *testing.T) {
	t.Parallel()

	adder := &fakeAdder{err: errors.New("disk full")}
	rec := postOrders(t, httpapi.New(adder, testLogger()),
		`{"customer_number":"C1","address":"1 High St","postcode":"AB1"}`, "application/json")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestPOSTOrders_panicRecoveredAs500(t *testing.T) {
	t.Parallel()

	adder := &fakeAdder{panicAdd: true}
	rec := postOrders(t, httpapi.New(adder, testLogger()),
		`{"customer_number":"C1","address":"1 High St","postcode":"AB1"}`, "application/json")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGETHealthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	httpapi.New(&fakeAdder{}, testLogger()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGETReadyz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	httpapi.New(&fakeAdder{}, testLogger()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type fakeAdder struct {
	mu       sync.Mutex
	orders   []order.Order
	err      error
	panicAdd bool
}

func (f *fakeAdder) Add(_ context.Context, o order.Order) error {
	if f.panicAdd {
		panic("adder panic")
	}
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	f.orders = append(f.orders, o)
	f.mu.Unlock()
	return nil
}

func (f *fakeAdder) addCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.orders)
}

func (f *fakeAdder) last() order.Order {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.orders[len(f.orders)-1]
}

func postOrders(t *testing.T, h http.Handler, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
