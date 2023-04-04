# s2x

`s2x` (sql-scan-x) is a small but opinionated lib to help scanning SQL rows into struct in Go.  

![build](https://github.com/MartyHub/s2x/actions/workflows/go.yml/badge.svg)

## Features

* No dependencies
* Compatible with `database/sql` package
* Use tag to map SQL column to struct
* Not recursive but accept multiple structs
* Check for duplicate SQL tags in struct or between structs
* Check for duplicate SQL columns in query
* Check for unmapped SQL columns
* Use cache for reflection

## Usage

```go
type MyStruct {
    MyField      string        `sql:my_field`
    OtherStruct  AnotherStruct
    privateField int
}

type AnotherStruct {
    AnotherField string `sql:another_field`
    privateField int
}

func scanAll(rows *sql.Rows) ([]MyStruct, error) {
    var result []MyStruct
	
    for scanner := s2x.NewScanner(rows); rows.Next(); {
        var myStruct MyStruct
    
        if err := scanner.Scan(&myStruct, &myStruct.AnotherStruct); err!=nil {
            return result, err
        }
    
        result := append(result, myStruct)
    }
    
    return result, rows.Err()
}
```
