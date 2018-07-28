// Package models contains the types for schema 'cyboard'.
package models

import (
	"database/sql/driver"
	"fmt"
)

// ExitStatus is the 'exit_status' enum type from schema 'cyboard'.
type ExitStatus uint16

const (
	// ExitStatusPass is the 'pass' ExitStatus.
	ExitStatusPass = ExitStatus(1)

	// ExitStatusFail is the 'fail' ExitStatus.
	ExitStatusFail = ExitStatus(2)

	// ExitStatusPartial is the 'partial' ExitStatus.
	ExitStatusPartial = ExitStatus(3)

	// ExitStatusTimeout is the 'timeout' ExitStatus.
	ExitStatusTimeout = ExitStatus(4)
)

// String returns the string value of the ExitStatus.
func (es ExitStatus) String() string {
	var enumVal string

	switch es {
	case ExitStatusPass:
		enumVal = "pass"

	case ExitStatusFail:
		enumVal = "fail"

	case ExitStatusPartial:
		enumVal = "partial"

	case ExitStatusTimeout:
		enumVal = "timeout"
	}

	return enumVal
}

// MarshalText marshals ExitStatus into text.
func (es ExitStatus) MarshalText() ([]byte, error) {
	return []byte(es.String()), nil
}

// UnmarshalText unmarshals ExitStatus from text.
func (es *ExitStatus) UnmarshalText(text []byte) error {
	switch string(text) {
	case "pass":
		*es = ExitStatusPass

	case "fail":
		*es = ExitStatusFail

	case "partial":
		*es = ExitStatusPartial

	case "timeout":
		*es = ExitStatusTimeout

	default:
		return fmt.Errorf("invalid ExitStatus %q", text)
	}

	return nil
}

// Value satisfies the sql/driver.Valuer interface for ExitStatus.
func (es ExitStatus) Value() (driver.Value, error) {
	return es.String(), nil
}

// Scan satisfies the database/sql.Scanner interface for ExitStatus.
func (es *ExitStatus) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("invalid ExitStatus '%v'", src)
	}

	return es.UnmarshalText([]byte(str))
}
