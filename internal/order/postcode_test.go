package order_test

import (
	"errors"
	"testing"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/order"
)

func TestValidatePostcode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		postcode string
		wantErr  error
	}{
		{name: "valid mixed case and digits", postcode: "SW1A 1A", wantErr: nil},
		{name: "valid single char", postcode: "A", wantErr: nil},
		{name: "valid max length eight", postcode: "AB12 3CD", wantErr: nil},
		{name: "valid digits only", postcode: "12345678", wantErr: nil},
		{name: "valid spaces only within limit", postcode: "  AB  ", wantErr: nil},

		{name: "empty", postcode: "", wantErr: order.ErrPostcodeEmpty},
		{name: "too long nine chars", postcode: "123456789", wantErr: order.ErrPostcodeTooLong},
		{name: "unicode letter", postcode: "SW1A\u00c9", wantErr: order.ErrPostcodeInvalidChar},
		{name: "hyphen", postcode: "SW1A-1", wantErr: order.ErrPostcodeInvalidChar},
		{name: "tab", postcode: "SW1\t1", wantErr: order.ErrPostcodeInvalidChar},
		{name: "newline", postcode: "SW1\n1", wantErr: order.ErrPostcodeInvalidChar},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := order.ValidatePostcode(tt.postcode)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidatePostcode(%q) error = %v, want %v", tt.postcode, err, tt.wantErr)
			}
		})
	}
}
