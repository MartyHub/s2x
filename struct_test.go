package s2x

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	Bool    bool   `sql:"bool"`
	Bytes   []byte `sql:"data"`
	Int     int
	String  string `sql:"string"`
	private string //nolint:unused
}

type testNestedStruct struct {
	MainBool   bool `sql:"mainBool"`
	MainInt    int
	MainString string `sql:"mainString"`
	Nested     testStruct
	Reader     io.Reader
	private    string //nolint:unused
}

type testNestedStructPointer struct {
	MainBool   bool `sql:"mainBool"`
	MainInt    int
	MainString string `sql:"mainString"`
	Pointer    *testStruct
	Reader     io.Reader
	private    string //nolint:unused
}

func TestStructColumns(t *testing.T) { //nolint:funlen
	type args struct {
		a          any
		exclusions []string
	}

	var data []byte

	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "struct",
			args: args{a: testStruct{}},
			want: map[string]any{
				"bool":   false,
				"data":   data,
				"string": "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "pointer on struct",
			args: args{a: &testStruct{}},
			want: map[string]any{
				"bool":   false,
				"data":   data,
				"string": "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "nested struct",
			args: args{a: testNestedStruct{}},
			want: map[string]any{
				"mainBool":   false,
				"mainString": "",
				"bool":       false,
				"data":       data,
				"string":     "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "pointer on nested struct",
			args: args{a: &testNestedStruct{}},
			want: map[string]any{
				"mainBool":   false,
				"mainString": "",
				"bool":       false,
				"data":       data,
				"string":     "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "nested struct pointer",
			args: args{a: testNestedStructPointer{}},
			want: map[string]any{
				"mainBool":   false,
				"mainString": "",
				"bool":       nil,
				"data":       nil,
				"string":     nil,
			},
			wantErr: assert.NoError,
		},
		{
			name: "pointer on nested struct pointer",
			args: args{a: &testNestedStructPointer{}},
			want: map[string]any{
				"mainBool":   false,
				"mainString": "",
				"bool":       nil,
				"data":       nil,
				"string":     nil,
			},
			wantErr: assert.NoError,
		},
		{
			name: "exclusions",
			args: args{
				a:          testNestedStruct{},
				exclusions: []string{"data", "mainString"},
			},
			want: map[string]any{
				"mainBool": false,
				"bool":     false,
				"string":   "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "exclusions with pointer",
			args: args{
				a:          &testNestedStruct{},
				exclusions: []string{"data", "mainString"},
			},
			want: map[string]any{
				"mainBool": false,
				"bool":     false,
				"string":   "",
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SQLValues(tt.args.a, tt.args.exclusions...)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
