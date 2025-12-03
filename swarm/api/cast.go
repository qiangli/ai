package api

import (
	"strings"
	"time"

	"github.com/spf13/cast"
)

// taken from viper@1.21.0
func (v ArgMap) Get(key string) any {
	lcaseKey := strings.ToLower(key)
	val, ok := v[lcaseKey]
	if ok {
		return val
	}
	return nil
}

func (v ArgMap) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

func (v ArgMap) GetBool(key string) bool {
	return cast.ToBool(v.Get(key))
}

func (v ArgMap) GetInt(key string) int {
	return cast.ToInt(v.Get(key))
}

func (v ArgMap) GetInt32(key string) int32 {
	return cast.ToInt32(v.Get(key))
}

func (v ArgMap) GetInt64(key string) int64 {
	return cast.ToInt64(v.Get(key))
}

func (v ArgMap) GetUint8(key string) uint8 {
	return cast.ToUint8(v.Get(key))
}

func (v ArgMap) GetUint(key string) uint {
	return cast.ToUint(v.Get(key))
}

func (v ArgMap) GetUint16(key string) uint16 {
	return cast.ToUint16(v.Get(key))
}

func (v ArgMap) GetUint32(key string) uint32 {
	return cast.ToUint32(v.Get(key))
}

func (v ArgMap) GetUint64(key string) uint64 {
	return cast.ToUint64(v.Get(key))
}

func (v ArgMap) GetFloat64(key string) float64 {
	return cast.ToFloat64(v.Get(key))
}

func (v ArgMap) GetTime(key string) time.Time {
	return cast.ToTime(v.Get(key))
}

func (v ArgMap) GetDuration(key string) time.Duration {
	return cast.ToDuration(v.Get(key))
}

func (v ArgMap) GetIntSlice(key string) []int {
	return cast.ToIntSlice(v.Get(key))
}

func (v ArgMap) GetStringSlice(key string) []string {
	return cast.ToStringSlice(v.Get(key))
}

func (v ArgMap) GetStringMap(key string) map[string]any {
	return cast.ToStringMap(v.Get(key))
}

func (v ArgMap) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(v.Get(key))
}

func (v ArgMap) GetStringMapStringSlice(key string) map[string][]string {
	return cast.ToStringMapStringSlice(v.Get(key))
}
