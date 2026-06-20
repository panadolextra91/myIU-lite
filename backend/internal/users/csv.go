package users

import (
	"encoding/csv"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	dobInputLayout    = "02/01/2006"
	dobPasswordLayout = "02012006"
)

func deriveDefaults(rawDOB string) (time.Time, string, error) {
	dob, err := time.Parse(dobInputLayout, rawDOB)
	if err != nil {
		return time.Time{}, "", ErrInvalidDOBFormat
	}

	pwStr := dob.Format(dobPasswordLayout)
	hash, err := bcrypt.GenerateFromPassword([]byte(pwStr), 12)
	if err != nil {
		return time.Time{}, "", err
	}

	return dob, string(hash), nil
}

func parseAccountsCSV(r io.Reader, idColumnName string) ([]ParsedAccount, []RowError) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, []RowError{{Row: 1, Field: "csv", Message: "failed to read headers"}}
	}

	// find columns
	idIdx, fnIdx, dobIdx := -1, -1, -1
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == idColumnName {
			idIdx = i
		} else if h == "full_name" {
			fnIdx = i
		} else if h == "dob" {
			dobIdx = i
		}
	}

	var rowErrs []RowError
	if idIdx == -1 {
		rowErrs = append(rowErrs, RowError{Row: 1, Field: idColumnName, Message: "missing column"})
	}
	if fnIdx == -1 {
		rowErrs = append(rowErrs, RowError{Row: 1, Field: "full_name", Message: "missing column"})
	}
	if dobIdx == -1 {
		rowErrs = append(rowErrs, RowError{Row: 1, Field: "dob", Message: "missing column"})
	}
	if len(rowErrs) > 0 {
		return nil, rowErrs
	}

	var parsed []ParsedAccount
	rowNum := 2
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: rowNum, Field: "csv", Message: "invalid row format"})
			rowNum++
			continue
		}

		id := strings.TrimLeft(strings.TrimSpace(record[idIdx]), "=+-@\t\r ")
		fn := strings.TrimLeft(strings.TrimSpace(record[fnIdx]), "=+-@\t\r ")
		dob := strings.TrimLeft(strings.TrimSpace(record[dobIdx]), "=+-@\t\r ")

		parsed = append(parsed, ParsedAccount{
			ID:       id,
			FullName: fn,
			DOB:      dob,
			RowIndex: rowNum,
		})
		rowNum++
	}

	return parsed, rowErrs
}

func parseStudentCSV(r io.Reader) ([]ParsedAccount, []RowError) {
	return parseAccountsCSV(r, "student_id")
}

func parseLecturerCSV(r io.Reader) ([]ParsedAccount, []RowError) {
	return parseAccountsCSV(r, "lecturer_id")
}
