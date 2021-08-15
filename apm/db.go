package apm

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// SetupDB setup DB info
func SetupDB(host, port, dbname string) {
	mySQLConnData = mySQLConnectionData{
		host:   host,
		port:   port,
		dbName: dbname,
	}
}

// DatastoreSegment (NoWeb)Datastore用セグメント
type DatastoreSegment struct {
	seg newrelic.DatastoreSegment
}

func (d DatastoreSegment) End() {
	if IsEnable() {
		d.seg.End()
	}
}

// StartDatastoreSegment (NoWeb)Datastore用セグメント開始
func StartDatastoreSegment(tx *Transaction, query string, params ...interface{}) DatastoreSegment {
	d := DatastoreSegment{}
	if tx.txn != nil && IsEnable() {
		d.seg = createDataStoreSegment(tx, query, params...)
	}
	return d
}

var (
	basicTable        = `[^)(\]\[\}\{\s,;]+`
	enclosedTable     = `[\[\(\{]` + `\s*` + basicTable + `\s*` + `[\]\)\}]`
	tablePattern      = `(` + `\s+` + basicTable + `|` + `\s*` + enclosedTable + `)`
	extractTableRegex = regexp.MustCompile(`[\s` + "`" + `"'\(\)\{\}\[\]]*`)
	updateRegex       = regexp.MustCompile(`(?is)^update(?:\s+(?:low_priority|ignore|or|rollback|abort|replace|fail|only))*` + tablePattern)
	sqlOperations     = map[string]*regexp.Regexp{
		"select":   regexp.MustCompile(`(?is)^.*?\sfrom` + tablePattern),
		"delete":   regexp.MustCompile(`(?is)^.*?\sfrom` + tablePattern),
		"insert":   regexp.MustCompile(`(?is)^.*?\sinto?` + tablePattern),
		"update":   updateRegex,
		"call":     nil,
		"create":   nil,
		"drop":     nil,
		"show":     nil,
		"set":      nil,
		"exec":     nil,
		"execute":  nil,
		"alter":    nil,
		"commit":   nil,
		"rollback": nil,
	}
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)(?:--|#).*?$`)
	sqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)
	mySQLConnData    = mySQLConnectionData{
		host:   "localhost",
		port:   "3306",
		dbName: "isucon",
	}
)

//queryはクエリ文。paramsにはクエリのパラメーターを可変長引数で渡す
func createDataStoreSegment(tx *Transaction, query string, params ...interface{}) newrelic.DatastoreSegment {
	queryParams := make(map[string]interface{}, len(params))
	var i = 0
	for _, param := range params {
		switch x := param.(type) {
		case []interface{}:
			for _, p := range x {
				queryParams["?_"+strconv.Itoa(i)] = p
				i++
			}
		case interface{}:
			queryParams["?_"+strconv.Itoa(i)] = x
			i++
		default:
			//ignore
		}
	}

	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = sqlPrefixRegex.ReplaceAllString(s, "")
	op := strings.ToLower(firstWordRegex.FindString(s))
	var operation, collection = "", ""
	if rg, ok := sqlOperations[op]; ok {
		operation = op
		if nil != rg {
			if m := rg.FindStringSubmatch(s); len(m) > 1 {
				collection = extractTable(m[1])
			}
		}
	}
	segment := newrelic.DatastoreSegment{
		StartTime:          tx.txn.StartSegmentNow(),
		Product:            newrelic.DatastoreMySQL,
		Collection:         collection,
		Operation:          operation,
		ParameterizedQuery: query,
		QueryParameters:    queryParams,
		Host:               mySQLConnData.host,
		PortPathOrID:       mySQLConnData.port,
		DatabaseName:       mySQLConnData.dbName,
	}
	return segment
}

//クエリからテーブル名と操作名をパースする処理はAgentのコードから流用
//the following code is copied from https://github.com/newrelic/go-agent/blob/06c801d5571056abac8ac9dfa07cf12ca869e920/v3/newrelic/sqlparse/sqlparse.go
func extractTable(s string) string {
	s = extractTableRegex.ReplaceAllString(s, "")
	if idx := strings.Index(s, "."); idx > 0 {
		s = s[idx+1:]
	}
	return s
}

type mySQLConnectionData struct {
	host   string
	port   string
	dbName string
}
