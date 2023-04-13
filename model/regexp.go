package model

import (
	"fmt"
	"regexp"
)

var (
	SplitterReg           = regexp.MustCompile(`-- Statement # \d+\n(--[^\n]*\n)*`)
	GeneratedReg          = regexp.MustCompile(`ADD GENERATED (BY DEFAULT|ALWAYS) AS IDENTITY \(\s*SEQUENCE NAME\s*.+START WITH \d+\s*INCREMENT BY \d+\s*NO MINVALUE\s*NO MAXVALUE\s*CACHE \d+\s*\);`)
	CreateTableCommentReg = regexp.MustCompile(`\n\s*-- count:(0|[1-9]\d{0,4})\n`)
	ArrayReg              = regexp.MustCompile(`\[[^\[\]\n\r]+,]`)
)

func GetColumnCommentReg(columnName string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`\n\s*(%s|"%s")[^\n\r]+ --[ ]*(type:[^\n\r]*|oneof:[^\n\r]*|range:[^\n\r]*)\n`, columnName, columnName))
}

func GetNthGroup(s string, reg *regexp.Regexp, n int) string {
	for i, match := range reg.FindStringSubmatch(s) {
		if i == n {
			return match
		}
	}

	return ""
}
