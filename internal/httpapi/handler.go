package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"orderservice/internal/order"
)

// MaxRequestBodyBytes is the maximum allowed POST /orders body size.
const MaxRequestBodyBytes = 64 << 10 // 64 KiB

// OrderAdder accepts a validated order into the batch buffer.
type OrderAdder interface {
	Add(ctx context.Context, o order.Order) error
}

type handler struct {
	adder  OrderAdder
	logger *slog.Logger
}

type createOrderRequest struct {
	CustomerNumber string     `json:"customer_number"`
	Address        string     `json:"address"`
	Postcode       string     `json:"postcode"`
	PlacedAt       *time.Time `json:"placed_at,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// New returns an HTTP handler with POST /orders, GET /healthz, and GET /readyz.
func New(adder OrderAdder, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{adder: adder, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.handleCreateOrder)
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /readyz", h.handleReadyz)
	return withMiddleware(mux, logger)
}

func (h *handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *handler) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			"Content-Type must be application/json")
		return
	}

	body := http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
	var req createOrderRequest
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		status := http.StatusBadRequest
		if isBodyTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, "INVALID_REQUEST", err.Error())
		return
	}

	placedAt := time.Now().UTC()
	if req.PlacedAt != nil {
		placedAt = req.PlacedAt.UTC()
	}

	o := order.Order{
		CustomerNumber: req.CustomerNumber,
		Address:        req.Address,
		Postcode:       req.Postcode,
		PlacedAt:       placedAt,
	}
	if err := o.Validate(); err != nil {
		code, msg := validationError(err)
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}

	if err := h.adder.Add(r.Context(), o); err != nil {
		h.logger.Error("add order failed", "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to accept order")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func isJSONContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return false
	}
	base, _, _ := strings.Cut(ct, ";")
	return strings.TrimSpace(base) == "application/json"
}

func isBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr) ||
		strings.Contains(err.Error(), "request body too large")
}

func validationError(err error) (code, message string) {
	switch {
	case errors.Is(err, order.ErrCustomerNumberRequired):
		return "CUSTOMER_NUMBER_REQUIRED", err.Error()
	case errors.Is(err, order.ErrAddressRequired):
		return "ADDRESS_REQUIRED", err.Error()
	case errors.Is(err, order.ErrPostcodeEmpty):
		return "POSTCODE_EMPTY", err.Error()
	case errors.Is(err, order.ErrPostcodeTooLong):
		return "POSTCODE_TOO_LONG", err.Error()
	case errors.Is(err, order.ErrPostcodeInvalidChar):
		return "POSTCODE_INVALID_CHAR", err.Error()
	default:
		return "VALIDATION_ERROR", err.Error()
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message, Code: code})
}
