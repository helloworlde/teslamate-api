package main

// based on Gist:
//   https://gist.github.com/rsudip90/022c4ef5d98130a224c9239e0a1ab397

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// NullInt64 可空整型，JSON 序列化为数字或 null（用于 API 响应字段）。
type NullInt64 struct {
	sql.NullInt64
}

// MarshalJSON for NullInt64
func (ni *NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// NullBool 可空布尔，JSON 序列化为 true/false 或 null。
type NullBool struct {
	sql.NullBool
}

// MarshalJSON for NullBool
func (nb *NullBool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nb.Bool)
}

// NullFloat64 可空浮点，JSON 序列化为数字或 null。
type NullFloat64 struct {
	sql.NullFloat64
}

// MarshalJSON for NullFloat64
func (nf *NullFloat64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

// NullString 可空字符串（数据库 NULL 映射为空串），JSON 省略空值语义与 sql 扫描配合使用。
type NullString string

func (s *NullString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	strVal, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}
	*s = NullString(strVal)
	return nil
}

func (s NullString) Value() (driver.Value, error) {
	if len(s) == 0 { // if nil or empty string
		return nil, nil
	}
	return string(s), nil
}
