package order_test

import (
	"errors"
	"testing"
	"time"

	"orderservice/internal/order"
)

func validOrder() order.Order {
	return order.Order{
		CustomerNumber: "CUST-001",
		Address:        "1 High Street, London",
		Postcode:       "SW1A 1AA",
		PlacedAt:       time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC),
	}
}

func TestOrderValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		order   order.Order
		wantErr error
	}{
		{name: "valid order", order: validOrder(), wantErr: nil},
		{
			name: "missing customer number",
			order: func() order.Order {
				o := validOrder()
				o.CustomerNumber = ""
				return o
			}(),
			wantErr: order.ErrCustomerNumberRequired,
		},
		{
			name: "whitespace only customer number",
			order: func() order.Order {
				o := validOrder()
				o.CustomerNumber = "   "
				return o
			}(),
			wantErr: order.ErrCustomerNumberRequired,
		},
		{
			name: "missing address",
			order: func() order.Order {
				o := validOrder()
				o.Address = ""
				return o
			}(),
			wantErr: order.ErrAddressRequired,
		},
		{
			name: "whitespace only address",
			order: func() order.Order {
				o := validOrder()
				o.Address = "\t"
				return o
			}(),
			wantErr: order.ErrAddressRequired,
		},
		{
			name: "empty postcode",
			order: func() order.Order {
				o := validOrder()
				o.Postcode = ""
				return o
			}(),
			wantErr: order.ErrPostcodeEmpty,
		},
		{
			name: "postcode too long",
			order: func() order.Order {
				o := validOrder()
				o.Postcode = "123456789"
				return o
			}(),
			wantErr: order.ErrPostcodeTooLong,
		},
		{
			name: "postcode invalid character",
			order: func() order.Order {
				o := validOrder()
				o.Postcode = "SW1A-1"
				return o
			}(),
			wantErr: order.ErrPostcodeInvalidChar,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.order.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
