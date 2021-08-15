package apm

import (
	"context"
	goji "goji.io"
	"goji.io/pat"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	app *newrelic.Application
	client = &http.Client{
		Transport: newrelic.NewRoundTripper(nil),
	}
	DefaultClient = http.DefaultClient // TODO: 問題によってはかわる可能性あり
)

// Transaction (NoWeb)トランザクション用
type Transaction struct {
	txn *newrelic.Transaction
}

func (t *Transaction) End() {
	if t.txn != nil {
		t.txn.End()
	}
}

// Setup APM設定
func Setup(appName string, license string) (err error) {
	if appName == "" || license == "" {
		return nil
	}

	app, err = newrelic.NewApplication(
		newrelic.ConfigAppName(appName),
		newrelic.ConfigLicense(license),
		newrelic.ConfigDistributedTracerEnabled(true),
		// newrelic.ConfigDebugLogger(os.Stdout),
	)
	return err
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
