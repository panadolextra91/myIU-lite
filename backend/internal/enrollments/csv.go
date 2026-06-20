package enrollments

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type ParsedRow struct {
	RowIndex int
	Username string
}

func ParseCSV(r io.Reader, role string) ([]ParsedRow, []RowError) {
	reader := csv.NewReader(r)
	
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, []RowError{{Row: 0, Field: "file", Message: "file is empty"}}
		}
		return nil, []RowError{{Row: 0, Field: "file", Message: "failed to read csv"}}
	}

	if len(header) == 0 {
		return nil, []RowError{{Row: 0, Field: "header", Message: "missing header"}}
	}

	expectedHeader := "student_id"
	if role == "lecturer" {
		expectedHeader = "lecturer_id"
	}

	idIdx := -1
	for i, h := range header {
		if strings.TrimSpace(strings.ToLower(h)) == expectedHeader {
			idIdx = i
			break
		}
	}

	if idIdx == -1 {
		return nil, []RowError{{Row: 1, Field: "header", Message: fmt.Sprintf("missing '%s' column", expectedHeader)}}
	}

	var parsed []ParsedRow
	var rowErrs []RowError
	seen := make(map[string]int)

	rowIndex := 2
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "row", Message: "malformed row"})
			rowIndex++
			continue
		}

		if len(record) <= idIdx {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: expectedHeader, Message: "missing value"})
			rowIndex++
			continue
		}

		username := strings.TrimSpace(record[idIdx])
		if username == "" {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: expectedHeader, Message: "empty value"})
			rowIndex++
			continue
		}

		if prev, exists := seen[username]; exists {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: expectedHeader, Message: fmt.Sprintf("duplicate ID in file (matches row %d)", prev)})
			rowIndex++
			continue
		}

		seen[username] = rowIndex
		parsed = append(parsed, ParsedRow{RowIndex: rowIndex, Username: username})
		rowIndex++
	}

	return parsed, rowErrs
}
