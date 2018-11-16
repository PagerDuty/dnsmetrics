package main

import (
	"reflect"
	"testing"
)

func TestSecondLastQps(t *testing.T) {
	t1 := map[string]float64{
		"foo": 1.,
		"bar": 2.,
	}
	t2 := map[string]float64{
		"foo": 3.,
		"bar": 4.,
	}
	t3 := map[string]float64{
		"bar": 5.,
		"baz": 6.,
	}

	raw := map[string]map[string]float64{
		"200": t1,
		"300": t2,
		"100": t3,
	}
	q := extractSecondLastQps(raw)
	if !reflect.DeepEqual(q, t1) {
		t.Error("Selected a wrong QPS dataset")
	}
}

func TestParseQpsCsv(t *testing.T) {
	input := `Head,Head2,Head3
1,foo,300
2,bar,600
1,bar,900`
	t1 := map[string]float64{
		"foo": 1.,
		"bar": 3.,
	}
	t2 := map[string]float64{
		"bar": 2.,
	}
	expected := map[string]map[string]float64{
		"1": t1,
		"2": t2,
	}

	out, err := parseQpsCsv(input)
	if err != nil {
		t.Error("Received error from parseQpsCsv ", err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Error("Output of parseQpsCsv does not match expectations")
	}
}
