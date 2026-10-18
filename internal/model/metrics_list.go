package model

import (
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// MetricsList — список метрик для пакетного обмена с агентом.
type MetricsList []Metrics

func (l MetricsList) MarshalEasyJSON(out *jwriter.Writer) {
	out.RawByte('[')
	for i := range l {
		if i > 0 {
			out.RawByte(',')
		}
		l[i].MarshalEasyJSON(out)
	}
	out.RawByte(']')
}

func (l MetricsList) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	l.MarshalEasyJSON(&w)
	return w.Buffer.BuildBytes(), w.Error
}

func (l *MetricsList) UnmarshalEasyJSON(in *jlexer.Lexer) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		in.Skip()
		*l = nil
	} else {
		in.Delim('[')
		if *l == nil {
			if !in.IsDelim(']') {
				*l = make(MetricsList, 0, 8)
			} else {
				*l = MetricsList{}
			}
		} else {
			*l = (*l)[:0]
		}
		for !in.IsDelim(']') {
			var metric Metrics
			metric.UnmarshalEasyJSON(in)
			*l = append(*l, metric)
			in.WantComma()
		}
		in.Delim(']')
	}
	if isTopLevel {
		in.Consumed()
	}
}

func (l *MetricsList) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	l.UnmarshalEasyJSON(&r)
	return r.Error()
}
