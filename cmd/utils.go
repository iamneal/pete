package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// returns the file, and the section after "option (persist.ql) = {" and right before the closing "}"
func protoFileQueriesPos(protoPath string) (file string, start, stop int, err error) {
	fbytes, err := ioutil.ReadFile(protoPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("error reading file '%s': \n%+v", protoPath, err)
	}

	file = string(fbytes)

	lineWithPersist := strings.Index(file, "persist.ql")
	if lineWithPersist < 0 {
		return "", 0, 0, fmt.Errorf("not a persist file")
	}
	// find next nearest newline, that is where we will start our search
	nextNl := strings.Index(file[lineWithPersist:], "\n")
	if nextNl < 0 {
		return "", 0, 0, fmt.Errorf("not a finished persist file")
	}

	// represents the number of '{' on the stack.  Every '{' rune incs, and '}' decs
	// goal being to get this to zero before eof, that is the queries we need to replace
	i := 0
	for braceStack := 1; braceStack != 0; i++ {
		c := file[lineWithPersist+nextNl+i]
		if c == '{' {
			braceStack++
		} else if c == '}' {
			braceStack--
		}
	}
	// line with opts, + offset of newline, + 1 to include the newline
	start = lineWithPersist + nextNl + 1
	// line with opts, + offset of newline, + till closing curly brace -1
	stop = lineWithPersist + nextNl + i - 1

	return
}

//
func peteQueriesFromFile(petePath, delimiter string) ([]string, error) {
	inbytes, err := ioutil.ReadFile(petePath)
	if err != nil {
		return nil, fmt.Errorf("error reading input file '%s': \n%+v", petePath, err)
	}
	return strings.Split(string(inbytes), delimiter), nil
}

// decorate each item in queries
func decoratePeteQueries(queries []string, linepad, prefix, tabsize string) {
	for i, v := range queries {
		if strings.TrimSpace(v) == "" {
			continue
		}
		queryParts := strings.Split(v, "\n")
		// TODO filter out bad lines
		serilizer := newQuerySerializer(queryParts, linepad, prefix)
		decoratedQuery := serilizer.Serialize(tabsize)
		// this isn't the last query so add a comma
		if i < len(queries)-1 {
			decoratedQuery += ","
		}
		queries[i] = decoratedQuery + "\n"
	}
}
