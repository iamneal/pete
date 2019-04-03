package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// returns the file, and the section after "option (persist.ql) = {" and right before the closing "}"
func protoFileQueriesPos(protoPath string) (file string, start, stop int, err error) {
	fbytes, err := ioutil.ReadFile(protoPath)
	if err != nil {
		return "", 0, 0, err
	}

	file = string(fbytes)

	lineWithPersist := strings.Index(file, "persist.ql")
	if lineWithPersist < 0 {
		return "", 0, 0, fmt.Errorf("Error: output is not a persist file")
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

func peteQueriesFromFile(petePath, delimiter string) ([]string, error) {
	inbytes, err := ioutil.ReadFile(petePath)
	if os.IsNotExist(err) {
		if _, err := os.Create(petePath); err != nil {
			return nil, err
		}
		inbytes, err = ioutil.ReadFile(petePath)
		if err != nil {
			return nil, err
		}
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
/* an example of one is this string
   queries: [
    {
      name: "GetCatByName",
      query: [
        "SELECT",
            "name,",
            "age,",
            "cost",
        "FROM cats",
        "WHERE",
            "name = @cat_name"
      ],
      pm_strategy: "$",
      in: ".test.CatName",
      out: ".test.Cat",

*/
type grossQuery string

func protoQueriesFromFile(protoPath string) ([]grossQuery, error) {
	file, start, end, err := protoFileQueriesPos(protoPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': \n+%v", protoPath, err)
	}
	queriesString := file[start:end]
	strs := strings.Split(queriesString, "},")
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
	queryParts := strings.Split(strings.TrimSpace(query), "\n")
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

func newQuerySerializerFromProto(query grossQuery, padding, prefix string) *querySerializer {
	queryStr := string(query)

	name, _ := namedCapture("name:( *)\"(?P<name>[a-zA-Z0-9]+)\"", queryStr)
	queryCapture, _ := namedCapture(`(?s)query:\s*\[(?P<query>[^\[\]]*)\]`, queryStr)
	inOutCapture, _ := namedCapture(`(?s)in:\s*"(?P<in>.*)",.*out:\s*"(?P<out>.*)"`, queryStr)

	q := new(querySerializer)
	q.padding = padding
	q.prefix = prefix
	q.name = name["name"]
	// TODO this is stupid.  Refactor pete to not include "in: " and "out: " and write it on serialization
	q.inline = "in: " + inOutCapture["in"]
	q.outline = "out: " + inOutCapture["out"]
	lineReg := regexp.MustCompile(`(?P<whitespace>\s*)"(?P<line>.+)"`)
	names := lineReg.SubexpNames()
	_ = names
	shortest := 999999999999
	lines := strings.Split(queryCapture["query"], "\n")

	// first pass get the shortest whitespace so we know how much to indent
	for _, line := range lines {
		subs := lineReg.FindStringSubmatch(line)
		if len(subs) < 2 {
			// probably an empty line or something
			continue
			// panic(fmt.Sprintf("query: %v\nline: %v\nsubs: %+v", queryCapture["query"], line, subs))
		}
		whitespace := len(subs[1])
		if shortest > whitespace {
			shortest = whitespace
		}
	}

	for _, line := range lines {
		subs := lineReg.FindStringSubmatch(line)
		if len(subs) < 2 {
			// probably an empty line or something
			continue
		}
		whitespace := len(subs[1]) - shortest
		currentLine := subs[2]
		q.query = append(q.query, makeSpaces(whitespace)+currentLine)
	}
	return q
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
	query := strings.Join(q.query, "\n")

	return strings.Join([]string{q.name, q.inline, q.outline, query}, "\n")
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

func makeSpaces(n int) string {
	arr := make([]string, n)
	for i := range arr {
		arr[i] = " "
	}
	return strings.Join(arr, "")
}

// performs the regular expression, an pulls out any named capture groups
func namedCapture(regXp, matchStr string) (map[string]string, *regexp.Regexp) {
	ret := make(map[string]string)

	compiledExp := regexp.MustCompile(regXp)
	matches := compiledExp.FindStringSubmatch(matchStr)
	for i, name := range compiledExp.SubexpNames() {
		// first item in matches needs to be ignored
		if i == 0 || i >= len(matches) || name == "" {
			continue
		}
		ret[name] = matches[i]
	}
	return ret, compiledExp
}
