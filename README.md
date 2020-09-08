# newrelic_apm

## Setup

```
	newrelicAppName := os.Getenv("NEW_RELIC_APP_NAME") // example: isucon9-qualify-muroon
	newrelicLicense := os.Getenv("NEW_RELIC_LICENSE")
	err = apm.Setup(newrelicAppName, newrelicLicense)
	if err != nil {
		log.Fatalf("failed to NewRelic: %s.", err.Error())
	}
```

## HTTP Handler

```
	apm.HandleFunc(mux, pat.Post("/initialize"), postInitialize)
```

## MiddlewareNewRelicTransaction
NewRelic Transactionのミドルウェアを設定

```
mux := goji.NewMux()
mux.Use(apm.MiddlewareNewRelicTransaction)

// API
mux.HandleFunc(pat.Post("/initialize"), postInitialize)
mux.HandleFunc(pat.Get("/new_items.json"), getNewItems)
(省略)
```

## RequestWithContext
http.RequestにNewRelicのコンテキストを付与
```
var (
	client = &http.Client{
		Transport: newrelic.NewRoundTripper(nil),
	}
)

(省略)

res, err := client.Do(apm.RequestWithContext(ctx, req))
```

## reference

https://github.com/muroon/isucon9-qualify/commit/9e0d5df64bd747288e1b49c1e680dd56dd75e771#diff-10a40f961254d187b7cb202a0c22bca0


