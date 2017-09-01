package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Bool string to bool
func Bool(f string) (bool, error) {
	return strconv.ParseBool(f)
}

// Float32 string to float32
func Float32(f string) (float32, error) {
	v, err := strconv.ParseFloat(f, 32)
	return float32(v), err
}

// Float64 string to float64
func Float64(f string) (float64, error) {
	return strconv.ParseFloat(f, 64)
}

// Int string to int
func Int(f string) (int, error) {
	v, err := strconv.ParseInt(f, 10, 32)
	return int(v), err
}

// Int8 string to int8
func Int8(f string) (int8, error) {
	v, err := strconv.ParseInt(f, 10, 8)
	return int8(v), err
}

// Int16 string to int16
func Int16(f string) (int16, error) {
	v, err := strconv.ParseInt(f, 10, 16)
	return int16(v), err
}

// Int32 string to int32
func Int32(f string) (int32, error) {
	v, err := strconv.ParseInt(f, 10, 32)
	return int32(v), err
}

// Int64 string to int64
func Int64(f string) (int64, error) {
	v, err := strconv.ParseInt(f, 10, 64)
	return int64(v), err
}

// Uint string to uint
func Uint(f string) (uint, error) {
	v, err := strconv.ParseUint(f, 10, 32)
	return uint(v), err
}

// Uint8 string to uint8
func Uint8(f string) (uint8, error) {
	v, err := strconv.ParseUint(f, 10, 8)
	return uint8(v), err
}

// Uint16 string to uint16
func Uint16(f string) (uint16, error) {
	v, err := strconv.ParseUint(f, 10, 16)
	return uint16(v), err
}

// Uint32 string to uint31
func Uint32(f string) (uint32, error) {
	v, err := strconv.ParseUint(f, 10, 32)
	return uint32(v), err
}

// Uint64 string to uint64
func Uint64(f string) (uint64, error) {
	v, err := strconv.ParseUint(f, 10, 64)
	return uint64(v), err
}

// ToStr interface to string
func ToStr(value interface{}, args ...int) (s string) {
	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 32))
	case float64:
		s = strconv.FormatFloat(v, 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 64))
	case int:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int8:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int16:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int32:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int64:
		s = strconv.FormatInt(v, argInt(args).Get(0, 10))
	case uint:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint8:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint16:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint32:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint64:
		s = strconv.FormatUint(v, argInt(args).Get(0, 10))
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		s = fmt.Sprintf("%v", v)
	}
	return s
}

// ToInt64 interface to int64
func ToInt64(value interface{}) (d int64) {
	val := reflect.ValueOf(value)
	switch value.(type) {
	case int, int8, int16, int32, int64:
		d = val.Int()
	case uint, uint8, uint16, uint32, uint64:
		d = int64(val.Uint())
	default:
		panic(fmt.Errorf("ToInt64 need numeric not `%T`", value))
	}
	return
}

// snake string, XxYy to xx_yy , XxYY to xx_yy
func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

// camel string, xx_yy to XxYy
func camelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

type argString []string

// get string by index from string slice
func (a argString) Get(i int, args ...string) (r string) {
	if i >= 0 && i < len(a) {
		r = a[i]
	} else if len(args) > 0 {
		r = args[0]
	}
	return
}

type argInt []int

// get int by index from int slice
func (a argInt) Get(i int, args ...int) (r int) {
	if i >= 0 && i < len(a) {
		r = a[i]
	}
	if len(args) > 0 {
		r = args[0]
	}
	return
}

var DefaultTimeLoc = time.Local

// parse time to string with location
func timeParse(dateString, format string) (time.Time, error) {
	tp, err := time.ParseInLocation(format, dateString, DefaultTimeLoc)
	return tp, err
}

// get pointer indirect type
func indirectType(v reflect.Type) reflect.Type {
	switch v.Kind() {
	case reflect.Ptr:
		return indirectType(v.Elem())
	default:
		return v
	}
}

func Interface2Map(value interface{}) (map[string]interface{}, error) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		return value.(map[string]interface{}), nil
	}
	return nil, errors.New("convert error")
}

func JsonStringToMap(jstr string) (map[string]interface{}, error) {
	resultMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(jstr), &resultMap); err != nil {
		return nil, err
	}
	return resultMap, nil
}
