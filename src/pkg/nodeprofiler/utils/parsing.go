package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ParsedOutput is a data structure that holds the parsed output
// of certain shell commands whose output takes the form of a
// table.
type ParsedOutput map[string][]string

// ParseColumns parses command outputs which are in a column table.
// It takes in a slice of strings which has all the column titles of the
// command in the correct order. This is necessary because some titles
// contain multiple strings, thus splitting by whitespaces will split row
// incorrectly. E.g, splitting the titles row in df's output based on
// whitespaces will result in:
//
// "Filesystem  Use% Mounted on" -> ["Filesystem", "Use%", "Mounted", "on"] instead of:
//                                  ["Filesystem", "Use%", "Mounted on"]
// The allTitles slice is used to give each title its correct index. If the
// titles in the slice are in wrong order, the function's output will be
// incorrect.
//
// The function also takes in an optional want titles slice which specifies the
// columns to parse. If this argument is missing, then all columns are parsed.
//
// Eg, ParseColumns(["r        b        swpd      buff",
//                  "10        0    14831128        0"],
//                  ["r", "b"]) = map[string][]string {
//                  "r": ["10"]
//                  "b": ["0"]}
//
// The output needs to have titles on all its columns else the function will
// return an error:
//
// Eg [FAIL] ParseColumns(["              total        used",
//                        "Mem:          14520          12",
//                        "Swap:             0           0"],
//                        ["total", "used"])
//                        err : "row has different number of columns from header row"
//
// Some edge cases that will be parsed by this function include:
//
// rows with repeated headers, eg with the output of iostat:
// [] string {"Device      tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
//            "vdb        2.39        57.39        69.83     855576    1041132",
//            "                                                               ",
//            "Device      tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
//            "                                                                ",
//            "Device      tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn"} =>
// map[string][]string {
// 	"tps" : {"2.39"},
//  "kB_read/s" : {"57.39"},
//  ...
// }
func ParseColumns(rows []string, allTitles []string, wantTitles ...string) (map[string][]string, error) {
	parsedOutput := ParsedOutput{}

	// maps each column title to its index eg columns["r"] = 0 wth vmstat.
	columns := make(map[string]int)
	// header saves the row that contains the titles
	var header string

	for i, row := range rows {
		// skip empty lines
		if row = strings.TrimSpace(row); row == "" {
			continue
		}

		if i == 0 {
			header = row
			// if no titles were specified, use all of them
			if len(wantTitles) == 0 {
				wantTitles = allTitles
			}
			// map each column title to its index eg "r" : 0 with vmstat
			for index, str := range allTitles {
				columns[str] = index
			}
			continue
		}
		// if a row is similar to the row with headers, ignore it
		if row == header {
			continue
		}

		tokens := strings.Fields(row)
		// checks that number of columns in row is equal to number of column
		// titles that were in header. Thus trying to parse the rows below
		// will return an error here:
		// "             total        used", (len 2)
		// "Mem:         14520          12", (len 3)
		// "Swap:            0           0",(len 3)
		if len(allTitles) != len(tokens) {
			err := "row has different number of columns from header row: \n" +
				"header row: \n %q \n " +
				"current row: \n %q"
			return nil, fmt.Errorf(err, header, row)
		}
		// loop over titles, get index of each title using map,
		// use that to get the actual values, add title and its value(s) to
		// parsedOutput.
		for _, title := range wantTitles {
			index, ok := columns[title]
			if !ok {
				return nil, fmt.Errorf("unknown column title %q", title)
			}
			// Get the actual value from the row. Eg, if 'r' is the title and
			// the lines to parse were below, then index = 0 and tokens[index] = 5
			// "r   b   swpd ..."
			// "5   0      0 ..."
			value := tokens[index]
			parsedOutput[title] = append(parsedOutput[title], value)
		}
	}
	return parsedOutput, nil
}

// ParseRows parses command outputs which are in a row table. It takes in a
// string which specifies the delimiter that separates row title from values.
// The function does not support '\n' as a delimiter for now. It takes in an
// optional titles slice which specifies which rows to parse.
//
// Eg, ParseRows(["avg-cpu:  %user %nice %system  %iowait %steal  %idle"],
//               [avg-cpu]) = map[string][]string {
//               avg-cpu: [%user, %nice, %system, %iowait, %steal, %idle]}
//
// If the wrong delimiter is passed in, the function returns an error:
//
// Eg [FAIL] ParseRows(["Device  tps   kB_read/s   kB_wrtn/s",
//                     "vdb 	1.13	  19.48 	  33.61"
//                     "vda    0.02       0.86       0.00"], ":", ["vda"])
//                     err: "failed to split row into row title and value"
//
// Some edge cases parsed by this function include:
//
// Rows whose delimiter have whitespaces around it. For example,
// [] string { "processor:     7",
//             "CPU family:    6"} =>
// map[string][]string {
//	  "processor"  : {"7"}
//    "cpu family" : {"6"}
// }
//
// OR
//
// [] string { "processor      : 7",
//             "cpu family     : 6"} =>
// map[string][]string {
// 	  "processor" : {"7"}
//    "cpu family" : {"6"}
// }
func ParseRows(lines []string, delim string, titles ...string) (map[string][]string, error) {
	parsedOutput := ParsedOutput{}
	// rows maps each row title to their value(s) which is the rest of the row
	// after delimiter
	rows := make(map[string][]string)
	for _, line := range lines {
		// skip empty lines
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		tokens := strings.Split(line, delim)
		// if row did not split, return an error.
		if len(tokens) == 1 {
			err := "failed to split %q by the delimiter %q"
			return nil, fmt.Errorf(err, line, delim)
		}
		// removes white space from title
		header := strings.TrimSpace(tokens[0])
		value := tokens[1:]
		// remove any extra white spaces in the values. Since values is a
		// slice, first join all the strings into 1 and split it. For example,
		//
		// "Architecture:     x86_64" will be split into ["Architecture", "    x86_64"]
		// To remove the whitespaces in [ "    x86_64"], join slice into
		// "x86_64" then split to make it a slice again ["x86_64"].
		tokens = strings.Fields(strings.Join(value, " "))
		rows[header] = tokens
	}
	// if no titles were passed, use all the row titles.
	if len(titles) == 0 {
		for key := range rows {
			titles = append(titles, key)
		}
	}
	// loop over titles passed to function (or initiliazed above), get their
	// values from the map, add to parsedOutput
	for _, title := range titles {
		var values []string
		var ok bool
		if values, ok = rows[title]; !ok {
			return nil, fmt.Errorf("unknown row title %q", title)
		}
		parsedOutput[title] = values
	}
	return parsedOutput, nil
}

// ParseRowsAndColumns parses command outputs that have row and column headers.
// It also takes in an optional titles slice which specifies the row column
// combination to parse. If the titles argument is missing, then an empty map
// is returned.
//
// Eg, ParseRowsAndColumns(["		total   used   free   shared",
//                         "Mem:   14520     12   14482        0",
//                         "Swap:      0      0       0        "],
//                         ["Mem:used", "Swap:total"]) = map[string][]string {
//                         "Mem:used": ["12"]
//                         "Swap:total" : ["0"]}
//
// The titles should be in the format "row:column". Else, an error is returned:
//
// Eg [FAIL], ParseRowsAndColumns(["       total   used   free   shared",
//                                  "Mem:   14520     12   14482       0",
//                                  "Swap:      0      0       0        "],
//                                  ["Mem+used", "Swap+total"])
//                                  err : "title string not well-formatted"
//
// Here are some edge cases parsed by the function:
//
// Rows with non-empty strings on row 0 column 0 E.g., with iostat (The default is
// an empty string on row 0 column 0):
// "Device             tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn",
// "vdb               0.74        10.39        23.23     859900    1922916",
// "vda               0.01         0.46         0.00      37829          0"
func ParseRowsAndColumns(lines []string, titles ...string) (map[string][]string, error) {
	parsedOutput := make(ParsedOutput)
	// columns maps column title to its index eg columns["total"] = 0 wth free.
	columns := make(map[string]int)
	// titlesMap maps a row title to columns titles based on the titles passed
	// into function:
	// Eg "Mem" : ["total", "used"] for "Mem:total", "Mem:used"
	//    "Swap": ["total", "used"] for "Swap:total", "Swap:used"
	titlesMap := make(map[string][]string)
	// loop over titles and split them by row and column titles.
	for _, title := range titles {
		headers := strings.Split(strings.Trim(title, ":"), ":")
		if length := len(headers); length == 2 {
			titlesMap[headers[0]] = append(titlesMap[headers[0]], headers[1])
		} else {
			err := "title string not well-formed: each title should " +
				"be in the form <row>:<column>, where row is the name " +
				"of the row header and column is the name of the " +
				"column header but got %q"
			return nil, fmt.Errorf(err, title)
		}
	}
	var diff int
	// rows stores each title in rows as key and the rest of the row as value.
	rows := make(map[string][]string)
	// loop over each row, mapping its title to the rest of the row (which is
	// its value).
	for i, line := range lines {
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}
		if i == 0 {
			// Looking at the edge case example above (iostat's output), since
			// rows are split by whitespaces, the index of "tps" will be 1
			// after split. When the second row is split, and divided into row
			// title and values, the following will result:
			// "vdb" : {"0.74", "10.39", "23.23", "859900", "1922916"}
			//
			// Index of column titles will be used to access values from slice
			// above. Index of "tps" = 1 and index 1 of slice above is 10.39
			// (which is incorrect). The correct value is in index 0 (which we
			// we would have gotten if col 0 row 0 was empty). To deal with this,
			// if column 0 of row 0 is a non-empty string, then 1 is subtracted
			// from the actual index of the rest of the colums in row 0. Thus the
			// need for the diff variable.
			exp := regexp.MustCompile(`\s*`)
			chars := exp.Split(line, -1)
			if chars[0] != "" {
				diff = -1
			}
			// map header name to its index
			for index, str := range tokens {
				columns[str] = index + diff
			}
			continue
		}
		rHeader := strings.Trim(tokens[0], ":")
		//everything to the right of the row title is its value
		rows[rHeader] = tokens[1:]
	}
	// loop over the titlesMap and use the row titles to access all
	// the values for that row. From those values, access the columns
	// we're interested in
	// Eg with free's output below:
	// "              total        used        free", (len 3)
	// "Mem:          14520          13       14482", (len 4)
	// "Swap:             0           0           0"  (len 4)
	//
	// Assuming the titlesMap is: "Mem"  : {"total", "used"}
	//						      "Swap" : {"total", "used"}
	//
	// When we loop over the map above, we first access the values for the
	// the row titles:  "Mem": {"14520", "13", "14482"}
	//                  "Swap": {"0", "0", "0"}
	// Then to access the values we're interested eg "Mem:total", use the index of
	// the column title "total" to index into the slice of values, i.e,
	// columns["total"] = 0 which corresponds to "14520" in {"14520", "13", "14482"}
	for rowTitle, colTitles := range titlesMap {
		values := rows[rowTitle]
		for _, columnTitle := range colTitles {
			index := columns[columnTitle]
			value := values[index]
			// combine the row and column title again when adding to the parsed
			// output map.
			combined := rowTitle + ":" + columnTitle
			parsedOutput[combined] = append(parsedOutput[combined], value)
		}
	}
	return parsedOutput, nil
}
