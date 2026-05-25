package order

import (
	"errors"
	"strings"
	"time"
)

// Order is a broadband service order received from an upstream service.
type Order struct {
	CustomerNumber string
	Address        string
	Postcode       string
	PlacedAt       time.Time
}

var (
	ErrCustomerNumberRequired = errors.New("customer number is required")
	ErrAddressRequired        = errors.New("address is required")
)

// Validate returns nil when all required fields are present and the postcode
// satisfies the take-home validation rule.
func (o Order) Validate() error {
	if strings.TrimSpace(o.CustomerNumber) == "" {
		return ErrCustomerNumberRequired
	}
	if strings.TrimSpace(o.Address) == "" {
		return ErrAddressRequired
	}
	return ValidatePostcode(o.Postcode)
}
