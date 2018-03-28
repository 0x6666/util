package cache

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"strconv"

	"github.com/0x6666/util/log"
)

// Serialize transforms the given value into bytes following these rules:
//   - If value is a byte array, it is returned as-is.
//   - If value is an int or uint type, it is returned as the ASCII representation
//   - Else, encoding/gob is used to serialize
func Serialize(value interface{}) ([]byte, error) {
	if data, ok := value.([]byte); ok {
		return data, nil
	}

	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	}

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	if err := encoder.Encode(value); err != nil {
		log.Error("Serialize: gob encoding failed value : %v, error: %v", value, err)
		return nil, err
	}
	return b.Bytes(), nil
}

// Deserialize transforms bytes produced by Serialize back into a Go object,
// storing it into "ptr", which must be a pointer to the value type.
func Deserialize(byt []byte, ptr interface{}) (err error) {
	if data, ok := ptr.(*[]byte); ok {
		*data = byt
		return
	}

	if v := reflect.ValueOf(ptr); v.Kind() == reflect.Ptr {
		switch p := v.Elem(); p.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var i int64
			i, err = strconv.ParseInt(string(byt), 10, 64)
			if err != nil {
				log.Error("Deserialize: failed to parse int value: %v, error: %v", string(byt), err)
			} else {
				p.SetInt(i)
			}
			return

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var i uint64
			i, err = strconv.ParseUint(string(byt), 10, 64)
			if err != nil {
				log.Error("Deserialize: failed to parse uint value: %v, error: ", string(byt), err)
			} else {
				p.SetUint(i)
			}
			return
		}
	}

	b := bytes.NewBuffer(byt)
	decoder := gob.NewDecoder(b)
	if err = decoder.Decode(ptr); err != nil {
		log.Error("Deserialize: glob decoding failed error: %v", err)
		return
	}
	return
}
