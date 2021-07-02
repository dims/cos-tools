package utils

import (
	"fmt"
	"strings"
)

// ParsedOutput is a data structure that holds the parsed output
// of certain shell commands whose output takes the form of a
// table.
type ParsedOutput map[string][]string

// ParseColumns parses command outputs which are in a column table.
// It takes in an optional titles argument which specifies the
// columns to parse. If this argument is missing, then all columns are parsed.
//
// Eg, ParseColumns(["total        used        free      shared",
// 					"14868916     12660    14830916          0"],
// 					["total", "used"]) = map[string][]string {
//					"total": ["14868916"]
//					"used": ["12660"]}
func ParseColumns(rows []string, titles ...string) (map[string][]string, error) {
	// parsedOutput is a map that stores col titles as key and entries as values.
	parsedOutput := ParsedOutput{}
	// maps column title to its index eg columns["r"] = 0 wth vmstat.
	columns := make(map[string]int)
	for i, row := range rows {
		// break the row into slice
		tokens := strings.Fields(row)
		if len(tokens) == 0 {
			continue
		}
		// find index of column titles
		if i == 0 {
			// if no titles were specified, use all of them
			if len(titles) == 0 {
				titles = tokens
			}
			// map header name to its index
			for index, str := range tokens {
				columns[str] = index
			}
			continue
		}
		// loop over titles, get column number of the title using map,
		// use that to get the actual values, append to list.
		for _, title := range titles {
			// for example columns["us"] = 12
			index, ok := columns[title]
			if !ok {
				return nil, fmt.Errorf("unknown Column title %s", title)
			}
			// for example if vmstat was run, tokens[0] will give the
			// value of a running process for some update.
			value := tokens[index]
			parsedOutput[title] = append(parsedOutput[title], value)
		}
	}
	return parsedOutput, nil
}

// ParseRows parses command outputs which are in a row table.
// It takes in an optional titles argument which specifies which rows
// to parse. If missing, all rows are parsed.
//
// Eg, ParseRows(["avg-cpu:  %user %nice %system  %iowait %steal  %idle"],
// 				 [avg-cpu]) = map[string][]string {
//				 avg-cpu: [%user, %nice, %system, %iowait, %steal, %idle]}
func ParseRows(lines []string, titles ...string) (map[string][]string, error) {
	// parsedOutput stores titles passed to function as key and entries as values.
	parsedOutput := ParsedOutput{}
	// rows stores each title in rows as key and the rest of the row as value.
	rows := make(map[string][]string)
	// loop over lines and map each line title to value(s)
	for _, line := range lines {
		// split by ':' for titles that are multi worded
		tokens := strings.Split(line, ":")
		// tokens is always at least of length 1 since an empty string when
		// split, will be a slice of length 1.
		if len(tokens) == 1 {
			continue
		}
		header := strings.Trim(tokens[0], "\\s*")
		// everything to the right of title is one string since
		// row was split by the character ':'.
		value := tokens[1]
		// now split value according to white spaces
		tokens = strings.Fields(value)
		rows[header] = tokens
	}
	// if empty titles slice was passed, use all the row titles.
	if len(titles) == 0 {
		for key := range rows {
			titles = append(titles, key)
		}
	}
	// loop over titles passed to function and get their values from the map.
	for _, title := range titles {
		var values []string
		var ok bool
		// check if any additional titles were passed in.
		if values, ok = rows[title]; !ok {
			return nil, fmt.Errorf("could not find the row title %s", title)
		}
		parsedOutput[title] = values
	}
	return parsedOutput, nil
}
