// Package models contains the types for schema 'cyboard'.
package models

import (
	"database/sql/driver"
	"fmt"
)

// TeamRole is the 'team_role' enum type from schema 'cyboard'.
type TeamRole uint16

const (
	// TeamRoleAdmin is the 'admin' TeamRole.
	TeamRoleAdmin = TeamRole(1)

	// TeamRoleCtfCreator is the 'ctf_creator' TeamRole.
	TeamRoleCtfCreator = TeamRole(2)

	// TeamRoleBlueteam is the 'blueteam' TeamRole.
	TeamRoleBlueteam = TeamRole(3)
)

// String returns the string value of the TeamRole.
func (tr TeamRole) String() string {
	var enumVal string

	switch tr {
	case TeamRoleAdmin:
		enumVal = "admin"

	case TeamRoleCtfCreator:
		enumVal = "ctf_creator"

	case TeamRoleBlueteam:
		enumVal = "blueteam"
	}

	return enumVal
}

// MarshalText marshals TeamRole into text.
func (tr TeamRole) MarshalText() ([]byte, error) {
	return []byte(tr.String()), nil
}

// UnmarshalText unmarshals TeamRole from text.
func (tr *TeamRole) UnmarshalText(text []byte) error {
	switch string(text) {
	case "admin":
		*tr = TeamRoleAdmin

	case "ctf_creator":
		*tr = TeamRoleCtfCreator

	case "blueteam":
		*tr = TeamRoleBlueteam

	default:
		return fmt.Errorf("invalid TeamRole %q", text)
	}

	return nil
}

// Value satisfies the sql/driver.Valuer interface for TeamRole.
func (tr TeamRole) Value() (driver.Value, error) {
	return tr.String(), nil
}

// Scan satisfies the database/sql.Scanner interface for TeamRole.
func (tr *TeamRole) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("invalid TeamRole '%v'", src)
	}

	return tr.UnmarshalText([]byte(str))
}
