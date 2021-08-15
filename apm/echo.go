package apm

import (
	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v3"
)

// MiddlewareNewRelicEcho echo middleware
func MiddlewareNewRelicEcho(e *echo.Echo) {
	if IsEnable() {
		e.Use(nrecho.Middleware(app))
	}
}

// TransactionFromEchoContext get Transaction from echo.Context
func TransactionFromEchoContext(c echo.Context) *Transaction {
	st := new(Transaction)
	if IsEnable() {
		st.txn = nrecho.FromContext(c)
	}
	return st
}
