package server

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// globRequired is the "Must" idiom for filepath.Glob
func mustGlobFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		Logger.Fatal(err)
	}
	return matches
}

func ReadStdinLine() ([]byte, error) {
	stdin := bufio.NewScanner(os.Stdin)
	stdin.Scan()
	return stdin.Bytes(), stdin.Err()
}

func sanitizeUpdateOp(updateOp map[string]interface{}) (map[string]interface{}, error) {
	if len(updateOp) == 0 {
		return nil, fmt.Errorf("no fields given for update: %v", updateOp)
	}
	sanitized := make(map[string]interface{}, len(updateOp))

	for k, v := range updateOp {
		// Check for empty strings
		if strValue, ok := v.(string); ok {
			if strValue == "" {
				return nil, fmt.Errorf("field must not be empty: %v", k)
			}
		}

		switch k {
		case "name":
			sanitized[k] = v.(string)
		case "group":
			sanitized[k] = v.(string)
		case "number":
			sanitized[k] = int64(v.(float64))
		case "ip":
			if parsed := net.ParseIP(v.(string)); parsed == nil {
				return nil, fmt.Errorf("invalid IP: %v", v)
			}
			sanitized[k] = v.(string)
		case "password":
			hashBytes, err := bcrypt.GenerateFromPassword([]byte(v.(string)), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("failed to hash password: %v", err)
			}
			sanitized["hash"] = string(hashBytes)
		default:
			return nil, fmt.Errorf("unexpected field in update JSON: %v: %v", k, v)
		}
	}

	return sanitized, nil
}

var TeamCsvHeaders = []string{"Name", "Group", "Number", "IP", "Password"}

func ParseTeamCsv(r io.Reader) ([]Team, error) {
	teamCsv := csv.NewReader(r)
	teamCsv.TrimLeadingSpace = true
	teamCsv.FieldsPerRecord = len(TeamCsvHeaders)

	posses := []Team{}
	rows, err := teamCsv.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read whole csv: %v", err)
	}

	if rows[0][0] == TeamCsvHeaders[0] {
		// Skip past the header line
		rows = rows[1:]
	}

	var parseErr error

	var rowIdx, colIdx int
	var row []string
	var column string

CsvParseLoop:
	for rowIdx, row = range rows {
		// Closure tracks the latest accessed column, to report in errors
		getColumn := func(idx int) string {
			colIdx = idx
			if idx > len(row) {
				return ""
			}
			return row[idx]
		}

		team := Team{}
		// Check for blank fields
		for colIdx, column = range row {
			if column == "" {
				parseErr = errors.New("must not be empty")
				break CsvParseLoop
			}
		}

		colIdx = 0

		team.Name = getColumn(0)
		team.Group = getColumn(1)

		if num, err := strconv.ParseInt(getColumn(2), 10, 0); err != nil {
			parseErr = fmt.Errorf("unable to parse Team Number: %v", err)
			break
		} else {
			team.Number = int(num)
		}

		if ip := net.ParseIP(getColumn(3)); ip == nil {
			if team.Group == "blueteam" {
				parseErr = fmt.Errorf("invalid IP: %v", getColumn(3))
				break
			}
		} else {
			team.Ip = ip.String()
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(getColumn(4)), bcrypt.DefaultCost)
		if err != nil {
			parseErr = fmt.Errorf("failed to hash password: %v", err)
			break
		}
		team.Hash = string(hashBytes)
		posses = append(posses, team)
	}

	if parseErr != nil {
		return nil, fmt.Errorf("error parsing csv - row #%d (%v), column %q: %v",
			rowIdx, row[0], TeamCsvHeaders[colIdx], parseErr)
	}

	return posses, nil
}
