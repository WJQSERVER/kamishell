package builtin

import (
	"os"
	"reflect"
	"time"
)

func currentFileTimes(info os.FileInfo) (time.Time, time.Time) {
	mtime := info.ModTime()
	atime := mtime

	sys := info.Sys()
	if sys == nil {
		return atime, mtime
	}

	if t, ok := extractFileTime(sys, "Atim", "Atimespec", "LastAccessTime"); ok {
		atime = t
	}
	if t, ok := extractFileTime(sys, "Mtim", "Mtimespec", "LastWriteTime"); ok {
		mtime = t
	}

	return atime, mtime
}

func extractFileTime(sys any, fieldNames ...string) (time.Time, bool) {
	v := reflect.ValueOf(sys)
	if !v.IsValid() {
		return time.Time{}, false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return time.Time{}, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return time.Time{}, false
	}

	for _, fieldName := range fieldNames {
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}
		if t, ok := reflectValueToTime(field); ok {
			return t, true
		}
	}

	return time.Time{}, false
}

func reflectValueToTime(v reflect.Value) (time.Time, bool) {
	if !v.IsValid() {
		return time.Time{}, false
	}

	if method := v.MethodByName("Nanoseconds"); method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 && method.Type().Out(0).Kind() == reflect.Int64 {
		out := method.Call(nil)
		return time.Unix(0, out[0].Int()), true
	}

	if v.Kind() != reflect.Struct {
		return time.Time{}, false
	}

	if sec, ok := reflectIntField(v, "Sec", "Tv_sec"); ok {
		nsec, _ := reflectIntField(v, "Nsec", "Tv_nsec")
		return time.Unix(sec, nsec), true
	}

	low := v.FieldByName("LowDateTime")
	high := v.FieldByName("HighDateTime")
	if low.IsValid() && high.IsValid() && isUnsigned(low.Kind()) && isUnsigned(high.Kind()) {
		ft := (high.Uint() << 32) | low.Uint()
		const windowsToUnixEpoch = 116444736000000000
		if ft >= windowsToUnixEpoch {
			return time.Unix(0, int64((ft-windowsToUnixEpoch)*100)), true
		}
	}

	return time.Time{}, false
}

func reflectIntField(v reflect.Value, names ...string) (int64, bool) {
	for _, name := range names {
		field := v.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return field.Int(), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return int64(field.Uint()), true
		}
	}
	return 0, false
}

func isUnsigned(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}
