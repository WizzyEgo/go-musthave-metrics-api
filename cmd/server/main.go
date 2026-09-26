package main

import (
	"go-musthave-metrics-tpl/internal/handler"
	"net/http"
)

func main() {
	err := http.ListenAndServe(":8080", handler.MetricHandler())

	if err != nil {
		panic(err)
	}
}
