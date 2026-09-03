package handler

import (
	"go-musthave-metrics-tpl/internal/model"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func MetricHandler() http.Handler {
	memStorage := model.NewMemStorage()
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", Update(memStorage))

	return mux
}

func Update(store model.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		metricType, metricName, metricValue, status := validateUrl(r)
		if status != 0 {
			w.WriteHeader(status)
			return
		}

		if err := updateMetric(store, metricType, metricName, metricValue); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	}
}

func validateUrl(r *http.Request) (metricType, metricName, metricValue string, status int) {
	parsedURL, err := url.Parse(r.URL.Path)
	if err != nil {
		return "", "", "", http.StatusBadRequest
	}

	segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(segments) != 4 {
		return "", "", "", http.StatusNotFound
	}

	metricType = segments[1]
	metricName = segments[2]
	metricValue = segments[3]

	if metricName == "" {
		return "", "", "", http.StatusNotFound
	}

	if metricType != "gauge" && metricType != "counter" {
		return "", "", "", http.StatusBadRequest
	}

	if metricType == "gauge" {
		if _, err := strconv.ParseFloat(metricValue, 64); err != nil {
			return "", "", "", http.StatusBadRequest
		}
	} else {
		if _, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			return "", "", "", http.StatusBadRequest
		}
	}

	return metricType, metricName, metricValue, 0
}

func updateMetric(store model.Storage, metricType, metricName, metricValue string) error {
	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		store.UpdateGauge(metricName, val)
	case "counter":
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}
		store.UpdateCounter(metricName, val)
	}
	return nil
}
