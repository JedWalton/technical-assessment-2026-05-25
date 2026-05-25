package order

import "errors"

const maxPostcodeLen = 8

var (
	ErrPostcodeEmpty       = errors.New("postcode is required")
	ErrPostcodeTooLong     = errors.New("postcode must be at most 8 characters")
	ErrPostcodeInvalidChar = errors.New("postcode may only contain letters, digits, and spaces")
)

// ValidatePostcode checks the postcode against the spec rule: 1–8
// characters of ASCII letters, digits, and spaces only.
func ValidatePostcode(postcode string) error {
	if postcode == "" {
		return ErrPostcodeEmpty
	}
	if len(postcode) > maxPostcodeLen {
		return ErrPostcodeTooLong
	}
	for _, r := range postcode {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == ' ':
		default:
			return ErrPostcodeInvalidChar
		}
	}
	return nil
}
