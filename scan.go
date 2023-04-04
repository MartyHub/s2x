package s2x

//go:generate mockery --name Rows

import (
	"database/sql"
	"reflect"
	"sync"
)

const SQLTag = "sql"

var (
	// SQLTagsCache cache SQL tags by struct type.
	SQLTagsCache = sync.Map{} //nolint:gochecknoglobals

	_ Rows = (*sql.Rows)(nil)
)

type Rows interface {
	Columns() ([]string, error)
	Scan(dest ...any) error
}

type Scanner struct {
	columns []columnMetadata
	rows    Rows
}

func NewScanner(rows Rows) *Scanner {
	return &Scanner{rows: rows}
}

func (scanner *Scanner) Scan(structs ...any) error {
	values, err := toValues(structs)
	if err != nil {
		return err
	}

	return scanner.doScan(values)
}

func (scanner *Scanner) doScan(values []reflect.Value) error {
	if scanner.columns == nil {
		if err := scanner.init(values); err != nil {
			return err
		}
	}

	return scanner.rows.Scan(scanner.newDest(values)...)
}

func (scanner *Scanner) newDest(values []reflect.Value) []any {
	result := make([]any, len(scanner.columns))

	for i, column := range scanner.columns {
		value := values[column.structIndex]

		result[i] = value.Field(column.fieldIndex).Addr().Interface()
	}

	return result
}

func (scanner *Scanner) init(values []reflect.Value) error {
	columns, err := scanner.rows.Columns()
	if err != nil {
		return err
	}

	columnsByName, err := computeMetadata(columns, values)
	if err != nil {
		return err
	}

	scanner.columns = make([]columnMetadata, len(columns))

	return scanner.doInit(columns, columnsByName)
}

func (scanner *Scanner) doInit(columns []string, columnsByName map[string]columnMetadata) error {
	uniq := make(map[string]bool, len(columns))

	for i, column := range columns {
		if uniq[column] {
			return newError("s2x: duplicate SQL column %q in query", column)
		}

		metadata, found := columnsByName[column]
		if !found {
			return newError("s2x: missing SQL tag for column %q", column)
		}

		scanner.columns[i] = metadata
		uniq[column] = true
	}

	return nil
}

type columnMetadata struct {
	structIndex int
	fieldIndex  int
}

// computeMetadata returns metadata by SQL tag in given structs, filtered by given columns.
func computeMetadata(columns []string, values []reflect.Value) (map[string]columnMetadata, error) {
	result := make(map[string]columnMetadata, len(columns))

	for structIndex, v := range values {
		indexByName, err := fieldsIndex(v.Type())
		if err != nil {
			return nil, err
		}

		for name, fieldIndex := range indexByName {
			if !contains(columns, name) {
				continue
			}

			if _, found := result[name]; found {
				return nil, newError("s2x: duplicate SQL tag %q between structs %v", name, values)
			}

			result[name] = columnMetadata{
				structIndex: structIndex,
				fieldIndex:  fieldIndex,
			}
		}
	}

	return result, nil
}

func fieldsIndex(t reflect.Type) (map[string]int, error) {
	cache, found := SQLTagsCache.Load(t)
	if found {
		return cache.(map[string]int), nil //nolint:forcetypeassert
	}

	result, err := buildFieldsIndex(t)
	if err != nil {
		return nil, err
	}

	SQLTagsCache.Store(t, result)

	return result, nil
}

// buildFieldsIndex returns field index by SQL tag in given struct type.
func buildFieldsIndex(t reflect.Type) (map[string]int, error) {
	result := make(map[string]int, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if tag, found := field.Tag.Lookup(SQLTag); found {
			if _, found := result[tag]; found {
				return nil, newError("s2x: duplicate SQL tag %q in %v", tag, t)
			}

			result[tag] = i
		}
	}

	return result, nil
}

func toValues(structs []any) ([]reflect.Value, error) {
	result := make([]reflect.Value, len(structs))

	for i, s := range structs {
		vs := reflect.ValueOf(s)

		if vs.Type().Kind() != reflect.Ptr || vs.IsNil() {
			return nil, newError("s2x: expected a non nil pointer, got %v", s)
		}

		result[i] = vs.Elem()

		if result[i].Type().Kind() != reflect.Struct {
			return nil, newError("s2x: expected a pointer on a struct, got %v", s)
		}
	}

	return result, nil
}
