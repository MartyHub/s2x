package s2x

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var testScanStructType = reflect.TypeOf(testScanStruct{}) //nolint:gochecknoglobals

type testScanStruct struct {
	Bool     bool    `sql:"bool"`
	Bytes    []byte  `sql:"data"`
	String   string  `sql:"string"`
	Nullable *string `sql:"nullable_string"`

	Unmapped int

	private string //nolint:unused
}

type testScanOtherStruct struct {
	Int      int  `sql:"number"`
	Nullable *int `sql:"nullable_int"`

	Unmapped bool

	private string //nolint:unused
}

type testScanStructSingleField struct {
	Bool bool `sql:"bool"`

	private string //nolint:unused
}

type testScanStructDuplicateField struct {
	Bool1 bool `sql:"bool"`
	Bool2 bool `sql:"bool"`

	private string //nolint:unused
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(`create table test_scan_struct (
		bool            integer not null,
    	data            blob not null,
    	string          text not null,
    	nullable_string text
	)`)

	return err
}

func insert(db *sql.DB, ts testScanStruct) error {
	result, err := db.Exec("insert into test_scan_struct (bool, data, string, nullable_string) values (?, ?, ?, ?)",
		ts.Bool,
		ts.Bytes,
		ts.String,
		ts.Nullable,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return errors.New("insert failed") //nolint:goerr113
	}

	return nil
}

func TestScanner_Scan(t *testing.T) {
	db := testDB(t)

	err := createTable(db)
	require.NoError(t, err)

	name := t.Name()

	expectedTS := testScanStruct{
		Bool:     true,
		Bytes:    []byte(t.Name()),
		Nullable: &name,
		String:   name,
	}
	err = insert(db, expectedTS)
	require.NoError(t, err)

	rows, err := db.Query("select bool, data, string, nullable_string from test_scan_struct")
	require.NoError(t, err)

	defer rows.Close()

	scanner := NewScanner(rows)
	require.True(t, rows.Next())

	ts := &testScanStruct{}

	require.NoError(t, scanner.Scan(ts))

	assert.Equal(t, expectedTS.Bool, ts.Bool)
	assert.Equal(t, expectedTS.Bytes, ts.Bytes)
	assert.Equal(t, expectedTS.Nullable, ts.Nullable)
	assert.Equal(t, expectedTS.String, ts.String)

	assert.NoError(t, rows.Err())
}

func TestScanner_doScan(t *testing.T) {
	db := testDB(t)

	err := createTable(db)
	require.NoError(t, err)

	expectedTS := testScanStruct{
		Bool:   true,
		Bytes:  []byte(t.Name()),
		String: t.Name(),
	}
	err = insert(db, expectedTS)
	require.NoError(t, err)

	rows, err := db.Query("select bool, data, string from test_scan_struct")
	require.NoError(t, err)

	defer rows.Close()

	scanner := NewScanner(rows)
	require.True(t, rows.Next())

	ts := &testScanStruct{}
	values := []reflect.Value{reflect.ValueOf(ts).Elem()}

	require.NoError(t, scanner.doScan(values))

	assert.Equal(t, expectedTS.Bool, ts.Bool)
	assert.Equal(t, expectedTS.Bytes, ts.Bytes)
	assert.Equal(t, expectedTS.Nullable, ts.Nullable)
	assert.Equal(t, expectedTS.String, ts.String)

	assert.NoError(t, rows.Err())
}

func TestScanner_init(t *testing.T) { //nolint:funlen
	mainStruct := &testScanStruct{}
	otherStruct := &testScanOtherStruct{}
	singleStruct := &testScanStructSingleField{}
	tests := []struct {
		name    string
		setup   func(rows *MockRows) *Scanner
		values  []reflect.Value
		want    []columnMetadata
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			setup: func(rows *MockRows) *Scanner {
				rows.EXPECT().Columns().Return([]string{"bool", "string", "number"}, nil)

				return NewScanner(rows)
			},
			values: []reflect.Value{reflect.ValueOf(mainStruct).Elem(), reflect.ValueOf(otherStruct).Elem()},
			want: []columnMetadata{
				{structIndex: 0, fieldIndex: 0},
				{structIndex: 0, fieldIndex: 2},
				{structIndex: 1, fieldIndex: 0},
			},
			wantErr: assert.NoError,
		},
		{
			name: "columns error",
			setup: func(rows *MockRows) *Scanner {
				rows.EXPECT().Columns().Return(nil, errors.New("test error")) //nolint:goerr113

				return NewScanner(rows)
			},
			values: []reflect.Value{reflect.ValueOf(mainStruct).Elem(), reflect.ValueOf(otherStruct).Elem()},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, "test error")
			},
		},
		{
			name: "duplicate SQL tag",
			setup: func(rows *MockRows) *Scanner {
				rows.EXPECT().Columns().Return([]string{"bool", "string", "mainBool", "mainString"}, nil)

				return NewScanner(rows)
			},
			values: []reflect.Value{reflect.ValueOf(singleStruct).Elem(), reflect.ValueOf(singleStruct).Elem()},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(
					t,
					err,
					`s2x: duplicate SQL tag "bool" between structs`+
						" [<s2x.testScanStructSingleField Value> <s2x.testScanStructSingleField Value>]",
				)
			},
		},
		{
			name: "duplicate SQL column",
			setup: func(rows *MockRows) *Scanner {
				rows.EXPECT().Columns().Return([]string{"bool", "bool"}, nil)

				return NewScanner(rows)
			},
			values: []reflect.Value{reflect.ValueOf(mainStruct).Elem()},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, `s2x: duplicate SQL column "bool" in query`)
			},
		},
		{
			name: "no SQL tag for column",
			setup: func(rows *MockRows) *Scanner {
				rows.EXPECT().Columns().Return([]string{"other"}, nil)

				return NewScanner(rows)
			},
			values: []reflect.Value{reflect.ValueOf(mainStruct).Elem()},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(
					t,
					err,
					`s2x: missing SQL tag for column "other"`,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := NewMockRows(t)
			scanner := tt.setup(rows)

			err := scanner.init(tt.values)
			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.want, scanner.columns)
			}
		})
	}
}

func TestScanner_newDest(t *testing.T) {
	rows := NewMockRows(t)
	scanner := NewScanner(rows)
	mainStruct := &testScanStruct{}
	otherStruct := &testScanOtherStruct{}
	values := []reflect.Value{reflect.ValueOf(mainStruct).Elem(), reflect.ValueOf(otherStruct).Elem()}

	rows.EXPECT().Columns().Return([]string{"bool", "string", "number", "nullable_int"}, nil)
	require.NoError(t, scanner.init(values))

	dest := scanner.newDest(values)

	assert.NotNil(t, dest)
	assert.Len(t, dest, 4)

	for i := 0; i < 4; i++ {
		assert.NotNil(t, dest[i])
	}

	assert.IsType(t, &mainStruct.Bool, dest[0])
	assert.IsType(t, &mainStruct.String, dest[1])
	assert.IsType(t, &otherStruct.Int, dest[2])
	assert.IsType(t, &otherStruct.Nullable, dest[3])
}

func Test_buildFieldsIndex(t *testing.T) {
	SQLTagsCache.Delete(testScanStructType)

	tests := []struct {
		name    string
		t       reflect.Type
		want    map[string]int
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			t:    testScanStructType,
			want: map[string]int{
				"bool":            0,
				"data":            1,
				"string":          2,
				"nullable_string": 3,
			},
			wantErr: assert.NoError,
		},
		{
			name:    "duplicate SQL tag",
			t:       reflect.TypeOf(testScanStructDuplicateField{}),
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFieldsIndex(tt.t)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fieldsIndex(t *testing.T) {
	SQLTagsCache.Delete(testScanStructType)

	tests := []struct {
		name    string
		t       reflect.Type
		want    map[string]int
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ok",
			t:    testScanStructType,
			want: map[string]int{
				"bool":            0,
				"data":            1,
				"string":          2,
				"nullable_string": 3,
			},
			wantErr: assert.NoError,
		},
		{
			name: "cache",
			t:    testScanStructType,
			want: map[string]int{
				"bool":            0,
				"data":            1,
				"string":          2,
				"nullable_string": 3,
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fieldsIndex(tt.t)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}
