package s2x

import (
	"reflect"
)

var (
	nilValuer         = func() any { return nil } //nolint:gochecknoglobals
	supportedSQLKinds = map[reflect.Kind]bool{    //nolint:gochecknoglobals
		reflect.Bool:      true,
		reflect.Int:       true,
		reflect.Int8:      true,
		reflect.Int16:     true,
		reflect.Int32:     true,
		reflect.Int64:     true,
		reflect.Uint:      true,
		reflect.Uint8:     true,
		reflect.Uint16:    true,
		reflect.Uint32:    true,
		reflect.Uint64:    true,
		reflect.Float32:   true,
		reflect.Float64:   true,
		reflect.String:    true,
		reflect.Interface: true,
	}
)

func SQLValues(a any, exclusions ...string) (map[string]any, error) {
	v, err := structValue(a)
	if err != nil {
		return nil, err
	}

	scanner := structSQLScanner{
		root:       v,
		exclusions: toMap(exclusions),
		sqlValues:  make(map[string]any),
	}

	err = scanner.ScanValue(v)

	return scanner.sqlValues, err
}

type structSQLScanner struct {
	root       any
	exclusions map[string]bool
	sqlValues  map[string]any
}

func (scanner structSQLScanner) ScanValue(v reflect.Value) error {
	var err error

	numField := v.NumField()

	for i := 0; i < numField; i++ {
		structField := v.Type().Field(i)

		if !structField.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		fieldType := fieldValue.Type()
		fieldKind := fieldType.Kind()

		for fieldKind == reflect.Ptr {
			fieldValue = fieldValue.Elem()
			fieldType = fieldType.Elem()
			fieldKind = fieldType.Kind()
		}

		if fieldKind == reflect.Struct {
			err = scanner.structFieldColumns(fieldValue, fieldType)
		} else {
			err = scanner.addValue(structField, fieldType, fieldValue.Interface)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (scanner structSQLScanner) scanType(v reflect.Type) error {
	var err error

	numField := v.NumField()

	for i := 0; i < numField; i++ {
		structField := v.Field(i)

		if !structField.IsExported() {
			continue
		}

		fieldType := structField.Type
		fieldKind := fieldType.Kind()

		for fieldKind == reflect.Ptr {
			fieldType = fieldType.Elem()
			fieldKind = fieldType.Kind()
		}

		if fieldKind == reflect.Struct {
			err = scanner.scanType(fieldType)
		} else {
			err = scanner.addValue(structField, fieldType, nilValuer)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (scanner structSQLScanner) structFieldColumns(fieldValue reflect.Value, fieldType reflect.Type) error {
	if fieldValue.IsValid() {
		return scanner.ScanValue(fieldValue)
	}

	return scanner.scanType(fieldType)
}

func (scanner structSQLScanner) addValue(
	structField reflect.StructField,
	fieldType reflect.Type,
	valuer func() any,
) error {
	if supportedColumnType(fieldType) {
		name := SQLName(structField)

		if name != "" && !scanner.exclusions[name] {
			if _, found := scanner.sqlValues[name]; found {
				return newError("s2x: duplicate SQL column %q in struct %v", name, scanner.root)
			}

			scanner.sqlValues[name] = valuer()
		}
	}

	return nil
}

func SQLName(field reflect.StructField) string {
	if tag, ok := field.Tag.Lookup(SQLTag); ok {
		return tag
	}

	return ""
}

func structValue(a any) (reflect.Value, error) {
	v := reflect.ValueOf(a)
	kind := v.Kind()

	for ; kind == reflect.Interface || kind == reflect.Ptr; kind = v.Kind() {
		v = v.Elem()
	}

	if kind != reflect.Struct {
		return v, newError("s2x: failed to find a struct in %v", a)
	}

	return v, nil
}

func supportedColumnType(t reflect.Type) bool {
	kind := t.Kind()

	return supportedSQLKinds[kind] || kind == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

func toMap(a []string) map[string]bool {
	result := make(map[string]bool, len(a))

	for _, item := range a {
		result[item] = true
	}

	return result
}
