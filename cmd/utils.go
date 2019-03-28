package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"unicode"
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
		// TODO filter out bad lines
		serilizer := newQuerySerializerFromPete(v, linepad, prefix)
		decoratedQuery := serilizer.ToProto(tabsize)
		// this isn't the last query so add a comma
		if i < len(queries)-1 {
			decoratedQuery += ","
		}
		queries[i] = decoratedQuery + "\n"
	}
}

// it has a bunch of garbage characters around it, but it
// contains 0 or 1 set of query options in its contents, but not more
type grossQuery string

func protoQueriesFromFile(protoPath string) ([]grossQuery, error) {
	pbytes, err := ioutil.ReadFile(protoPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': \n+%v", protoPath, err)
	}

	strs := strings.Split(string(pbytes), "},")
	out := make([]grossQuery, len(strs))

	for i, v := range strs {
		out[i] = grossQuery(v)
	}

	return out, nil
}

type querySerializer struct {
	padding string
	prefix  string
	name    string
	inline  string
	outline string
	query   []string
}

func newQuerySerializerFromPete(query string, padding, prefix string) *querySerializer {
	queryParts := strings.Split(query, "\n")
	q := new(querySerializer)
	// first line is always the name
	q.name = queryParts[0]
	queryParts = queryParts[1:]

	q.padding = padding
	q.prefix = prefix
	// the line that contains "in: " is the input
	// the line taht contains "out: " is the output
	for _, v := range queryParts {
		if strings.HasPrefix(v, "in: ") {
			q.inline = strings.TrimPrefix(v, "in: ")
		} else if strings.Contains(v, "out: ") {
			q.outline = strings.TrimPrefix(v, "out: ")
		} else {
			q.query = append(q.query, v)
		}
	}
	return q
}

// TODO
func newQuerySerializerFromProto(query grossQuery, padding, prefix string) *querySerializer {
	q := new(querySerializer)
	q.padding = padding
	q.prefix = prefix
	fmt.Printf("my query: \n%+v\n\n", query)

	return nil
}

/*
	padding + { + \n
	padding + \t + name: + " + VALUE + ", + \n
	padding + \t + query: [ + \n + VALUE + ], + \n
	padding + \t + " + prefix + VALUE + ", + \n
	padding + \t + pm_strategy: + " + $ + "
	padding + } + ,
*/
func (q *querySerializer) ToProto(tabsize string) string {
	var decoratedQuery, squashed string
	comma := func(i int) string {
		if i == len(q.query)-1 {
			return ""
		}
		return ","
	}
	typename := func(s string) string {
		// we must already have a full annotated path, so no adjustment
		if strings.Contains(s, ".") || q.prefix == "" {
			return s
		}
		return q.prefix + "." + s
	}

	queryStringTab := tabsize + tabsize
	insideBraceTab := tabsize + tabsize + tabsize
	for i, v := range q.query {
		// we need to make this line pretty
		spaces, line := trimLeftAndKeepSpaces(v)
		squashed += q.padding + queryStringTab + spaces + fmt.Sprintf(`"%s"`, line) + comma(i) + "\n"
	}

	// TODO make }, { optional
	decoratedQuery += q.padding + "{\n"
	// TODO make the tab size adjustable
	decoratedQuery += insideBraceTab + fmt.Sprintf(`name: "%s",`+"\n", q.name)
	decoratedQuery += insideBraceTab + "query: [\n"
	decoratedQuery += squashed
	decoratedQuery += insideBraceTab + "],\n"
	// TODO make this optional too
	decoratedQuery += insideBraceTab + `pm_strategy: "$",` + "\n"
	decoratedQuery += insideBraceTab + fmt.Sprintf(`in: "%s",`+"\n", typename(q.inline))
	decoratedQuery += insideBraceTab + fmt.Sprintf(`out: "%s",`+"\n", typename(q.outline))
	decoratedQuery += q.padding + "}"

	return decoratedQuery
}

func (q *querySerializer) ToPete() string {
	return ""
}

func (q *querySerializer) Undecorate() (name, in, out string, query []string) {
	return
}

func header(padding string) string {
	return padding + "queries: [\n"
}
func footer(padding string) string {
	return padding + "];\n"
}

func trimLeftAndKeepSpaces(s string) (spaces string, trimmed string) {
	trimmed = strings.TrimLeftFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			spaces += string(r)
			return true
		}
		return false
	})
	return
}
