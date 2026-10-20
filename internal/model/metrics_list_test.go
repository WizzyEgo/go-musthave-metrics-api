package model

import (
	"testing"

	"github.com/mailru/easyjson"
)

func TestMetricsListRoundTrip(t *testing.T) {
	value := 0.0
	delta := int64(0)
	original := MetricsList{
		{ID: "Alloc", MType: Gauge, Value: &value},
		{ID: "PollCount", MType: Counter, Delta: &delta},
	}

	payload, err := easyjson.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(payload) != `[{"id":"Alloc","type":"gauge","value":0},{"id":"PollCount","type":"counter","delta":0}]` {
		t.Fatalf("payload = %s", payload)
	}

	var got MetricsList
	if err := easyjson.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 || got[0].Value == nil || *got[0].Value != 0 || got[0].Delta != nil {
		t.Fatalf("gauge = %+v", got[0])
	}
	if got[1].Delta == nil || *got[1].Delta != 0 || got[1].Value != nil {
		t.Fatalf("counter = %+v", got[1])
	}
}

func TestMetricsListEmptyAndNull(t *testing.T) {
	var empty MetricsList
	if err := easyjson.Unmarshal([]byte(`[]`), &empty); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if empty == nil {
		t.Fatal("empty array must unmarshal to a non-nil list")
	}
	payload, err := easyjson.Marshal(MetricsList(nil))
	if err != nil {
		t.Fatalf("marshal nil: %v", err)
	}
	if string(payload) != "[]" {
		t.Fatalf("nil list payload = %s, want []", payload)
	}

	var nullList MetricsList
	if err := easyjson.Unmarshal([]byte(`null`), &nullList); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if nullList != nil {
		t.Fatalf("null list = %#v, want nil", nullList)
	}
}
