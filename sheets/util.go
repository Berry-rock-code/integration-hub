package sheets

import (
	"fmt"
	"strings"
)

func FindHeaderIndex(headers []string, want string) int {
	want = strings.TrimSpace(strings.ToLower(want))
	for i, h := range headers {
		if strings.TrimSpace(strings.ToLower(h)) == want {
			return i
		}
	}
	return -1
}

// ParseHeaderRow converts a row of interface{} into []string.
func ParseHeaderRow(row []interface{}) []string {
	out := make([]string, 0, len(row))
	for _, v := range row {
		if v == nil {
			out = append(out, "")
			continue
		}
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

func cellA1(col int, row int) string {
	// col: 1-based, row: 1-based
	// A=1, B=2 ...
	if col <= 0 || row <= 0 {
		return "A1"
	}
	colName := ""
	for col > 0 {
		col-- // 0-based
		colName = string(rune('A'+(col%26))) + colName
		col /= 26
	}
	return fmt.Sprintf("%s%d", colName, row)
}
