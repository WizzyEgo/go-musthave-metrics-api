package handler

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"

	"go-musthave-metrics-tpl/internal/storage"
)

type metricItem struct {
	Name  string
	Value string
}

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Metrics</title>
</head>
<body>
	<h1>Metrics</h1>
	<ul>
	{{range .}}
		<li>{{.Name}}: {{.Value}}</li>
	{{end}}
	</ul>
</body>
</html>`))

func Index(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges, counters := store.GetAll()
		items := make([]metricItem, 0, len(gauges)+len(counters))

		for name, value := range gauges {
			items = append(items, metricItem{
				Name:  name,
				Value: strconv.FormatFloat(value, 'f', -1, 64),
			})
		}
		for name, value := range counters {
			items = append(items, metricItem{
				Name:  name,
				Value: strconv.FormatInt(value, 10),
			})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexTmpl.Execute(w, items)
	}
}
