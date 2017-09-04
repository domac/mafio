package util

import (
	"reflect"
)

func Interface2Stringslice(arg interface{}) (out []string, ok bool) {
	//类型转换
	slice, success := convertArg(arg, reflect.Slice)
	if !success {
		ok = false
		return
	}

	c := slice.Len()
	out = make([]string, c)
	for i := 0; i < c; i++ {
		tmp := slice.Index(i).Interface()
		if tmp != nil {
			//强制转换为字符串
			out[i] = tmp.(string)
		}
	}

	return out, true
}

func convertArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}
