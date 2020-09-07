# newrelic_apm

## set up

```
	newrelicAppName := os.Getenv("NEW_RELIC_APP_NAME") // example: isucon9-qualify-muroon
	newrelicLicense := os.Getenv("NEW_RELIC_LICENSE")
	err = apm.Setup(newrelicAppName, newrelicLicense)
	if err != nil {
		log.Fatalf("failed to NewRelic: %s.", err.Error())
	}
```


## handler

```
	apm.HandleFunc(mux, pat.Post("/initialize"), postInitialize)
```

## reference

https://github.com/muroon/isucon9-qualify/commit/9e0d5df64bd747288e1b49c1e680dd56dd75e771#diff-10a40f961254d187b7cb202a0c22bca0


