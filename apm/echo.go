package apm

import (
	"github.com/labstack/echo"
	echov4 "github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v3"
	nrechov4 "github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
)

// MiddlewareNewRelicEcho echo middleware (for v3)
func MiddlewareNewRelicEcho(e *echo.Echo) {
	if IsEnable() {
		e.Use(nrecho.Middleware(app))
	}
}

// TransactionFromEchoContext get Transaction from echo.Context (for v3)
func TransactionFromEchoContext(c echo.Context) *Transaction {
	st := new(Transaction)
	if IsEnable() {
		st.txn = nrecho.FromContext(c)
	}
	return st
}

// MiddlewareNewRelicEchoV4 echo middleware (for v4)
func MiddlewareNewRelicEchoV4(e *echov4.Echo) {
	if IsEnable() {
		e.Use(nrechov4.Middleware(app))
	}
}

// TransactionFromEchoContextV4 get Transaction from echo.Context (for v4)
func TransactionFromEchoContextV4(c echov4.Context) *Transaction {
	st := new(Transaction)
	if IsEnable() {
		st.txn = nrechov4.FromContext(c)
	}
	return st
}
