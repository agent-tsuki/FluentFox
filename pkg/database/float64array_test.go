package database

import (
	"reflect"
	"testing"
)

func TestFloat64ArrayValue(t *testing.T) {
	tests := []struct {
		name string
		in   Float64Array
		want any
	}{
		{name: "nil writes NULL", in: nil, want: nil},
		{name: "empty writes an empty array", in: Float64Array{}, want: "{}"},
		{name: "values keep full precision", in: Float64Array{0.212, 1.2931, 2}, want: "{0.212,1.2931,2}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.Value()
			if err != nil || got != tc.want {
				t.Fatalf("Value() = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestFloat64ArrayScan(t *testing.T) {
	tests := []struct {
		name string
		src  any
		want Float64Array
	}{
		{name: "NULL reads as nil", src: nil, want: nil},
		{name: "a text literal is parsed", src: "{0.5,1.25}", want: Float64Array{0.5, 1.25}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Float64Array
			if err := got.Scan(tc.src); err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Scan = %#v, want %#v", got, tc.want)
			}
		})
	}
}
