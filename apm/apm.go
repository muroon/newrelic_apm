package apm

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
	goji "goji.io"
	"goji.io/pat"
)

var (
	app    *newrelic.Application
	client = &http.Client{
		Transport: newrelic.NewRoundTripper(nil),
	}
	mySQLConnData = mySQLConnectionData{
		host:   "localhost",
		port:   "3306",
		dbName: "isucon",
	}
	DefaultClient = http.DefaultClient // TODO: 問題によってはかわる可能性あり

)

type mySQLConnectionData struct {
	host   string
	port   string
	dbName string
}

// Transaction (NoWeb)トランザクション用
type Transaction struct {
	txn *newrelic.Transaction
}

func (t *Transaction) End() {
	if t.txn != nil {
		t.txn.End()
	}
}

// DatastoreSegment (NoWeb)Datastore用セグメント
type DatastoreSegment struct {
	seg newrelic.DatastoreSegment
}

func (d DatastoreSegment) End() {
	if isEnable() {
		d.seg.End()
	}
}

// Setup APM設定
func Setup(appName string, license string) (err error) {
	if appName == "" || license == "" {
		return errors.New("appName or Licence is empty")
	}

	app, err = newrelic.NewApplication(
		newrelic.ConfigAppName(appName),
		newrelic.ConfigLicense(license),
		newrelic.ConfigDistributedTracerEnabled(true),
		// newrelic.ConfigDebugLogger(os.Stdout),
	)
	return err
}

// SetupDB setup DB info
func SetupDB(host, port, dbname string) {
	mySQLConnData = mySQLConnectionData{
		host:   host,
		port:   port,
		dbName: dbname,
	}
}

// HandleFunc Web handler 設定
// TODO:問題によっては引数が変わる
func HandleFunc(mux *goji.Mux, pattern *pat.Pattern, hdl http.HandlerFunc) {
	whdl := hdl
	if isEnable() {
		_, whdl = newrelic.WrapHandleFunc(app, pattern.String(), hdl)
	}
	mux.HandleFunc(pattern, whdl)
}

// Handle Web handler 設定
// TODO:問題によっては引数が変わる
func Handle(mux *goji.Mux, pattern *pat.Pattern, hdl http.Handler) {
	whdl := hdl
	if isEnable() {
		_, whdl = newrelic.WrapHandle(app, pattern.String(), hdl)
	}
	mux.Handle(pattern, whdl)
}

// StartTransaction (NoWeb)トランザクション開始
func StartTransaction(name string) *Transaction {
	st := new(Transaction)
	if isEnable() {
		st.txn = app.StartTransaction(name)
	}
	return st
}

// StartDatastoreSegment (NoWeb)Datastore用セグメント開始
func StartDatastoreSegment(tx *Transaction, query string, params ...interface{}) DatastoreSegment {
	d := DatastoreSegment{}
	if tx.txn != nil && isEnable() {
		d.seg = createDataStoreSegment(tx, query, params...)
	}
	return d
}

// createDataStoreSegmentで使用する変数
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

func isEnable() bool {
	return app != nil
}

// MiddlewareNewRelicTransaction to create/end NewRelic transaction
func MiddlewareNewRelicTransaction(inner http.Handler) http.Handler {
	if !isEnable() {
		mw := func(w http.ResponseWriter, r *http.Request) {
			inner.ServeHTTP(w, r)
		}
		return http.HandlerFunc(mw)
	}
	mw := func(w http.ResponseWriter, r *http.Request) {
		txn := app.StartTransaction(r.URL.Path)
		defer txn.End()

		r = newrelic.RequestWithTransactionContext(r, txn)
		txn.SetWebRequestHTTP(r)
		w = txn.SetWebResponse(w)
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// RequestWithContext RequestにNewRelicのContextをつける
func RequestWithContext(ctx context.Context, req *http.Request) *http.Request {
	if !isEnable() {
		return req
	}
	txn := newrelic.FromContext(ctx)
	req = newrelic.RequestWithTransactionContext(req, txn)
	return req
}

// GetClient リクエスト送信クライアントを返す
func GetClient() *http.Client {
	if !isEnable() {
		return DefaultClient
	}

	return client
}

// RequestDoWithContext リクエスト送信クライアントの振り分けとRequestにNewRelicのContextをつけて送信を行う
func RequestDoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = RequestWithContext(ctx, req)
	return GetClient().Do(req)
}
