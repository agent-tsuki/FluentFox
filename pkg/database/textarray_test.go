package database

import (
	"reflect"
	"testing"
)

func TestTextArrayValue(t *testing.T) {
	tests := []struct {
		name string
		in   TextArray
		want string
	}{
		{name: "nil writes an empty array", in: nil, want: "{}"},
		{name: "elements are quoted", in: TextArray{"one", "one radical (no.1)"}, want: `{"one","one radical (no.1)"}`},
		{name: "quotes and backslashes are escaped", in: TextArray{`a"b`, `c\d`}, want: `{"a\"b","c\\d"}`},
		{name: "NULL and empty stay strings", in: TextArray{"NULL", ""}, want: `{"NULL",""}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.Value()
			if err != nil || got != tc.want {
				t.Fatalf("Value() = %v, %v; want %s", got, err, tc.want)
			}
		})
	}
}

func TestTextArrayScan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		want    TextArray
		wantErr bool
	}{
		{name: "NULL scans to an empty array", src: nil, want: TextArray{}},
		{name: "text literal from a string", src: `{one,"two words",ニチ}`, want: TextArray{"one", "two words", "ニチ"}},
		{name: "text literal from bytes", src: []byte(`{"a\"b"}`), want: TextArray{`a"b`}},
		{name: "a value written by Value reads back the same", src: `{"NULL",""}`, want: TextArray{"NULL", ""}},
		{name: "a non-array value is an error", src: 42, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got TextArray
			err := got.Scan(tc.src)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Scan() = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}
