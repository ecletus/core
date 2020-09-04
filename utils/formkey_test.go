package utils

import (
	"reflect"
	"testing"
)

func TestParseFormKey(t *testing.T) {
	tests := []struct {
		key string
		wantResult []interface{}
		wantErr    string
	}{
		{"a[1][x].v", []interface{}{"a", 1, "x", "v"}, ""},
		{"a[1][x]v", []interface{}{"a", 1, "x", "v"}, ""},
		{"a[001][x]v", []interface{}{"a", 1, "x", "v"}, ""},
		{"a[1[x]v", []interface{}{}, "malformed key: un expected char key[3] = '['"},
		{"a[1][x.v", []interface{}{}, "malformed key: unclosed index name started at key[4]"},
		{"a1][x.v", []interface{}{}, "malformed key: un expected key[2] = ']'"},
		{"a[1][x.v]", []interface{}{"a", 1, "x.v"}, ""},
		{"a[1][2][x.v]", []interface{}{"a", 1, 2, "x.v"}, ""},
		{"a[1][][x.v]", []interface{}{"a", 1, -1, "x.v"}, ""},
		{"a[][x.v]", []interface{}{"a", -1, "x.v"}, ""},

	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotResult, err := ParseFormKey(tt.key)
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Errorf("ParseFormKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("ParseFormKey() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
