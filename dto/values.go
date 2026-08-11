package dto

import (
	"encoding/json"
	"strconv"
	"strings"
)

type IntValue int

func (i *IntValue) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*i = IntValue(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = IntValue(v)
	return nil
}

func (i IntValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(i))
}

type BoolValue bool

func (b *BoolValue) UnmarshalJSON(data []byte) error {
	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		*b = BoolValue(boolean)
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	// 兼容 multipart/form 与部分客户端把 bool 写成字符串（"true"/"1"/"TRUE" 等）
	parsed, err := strconv.ParseBool(strings.TrimSpace(str))
	if err != nil {
		return err
	}
	*b = BoolValue(parsed)
	return nil
}

func (b BoolValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}

// BoolPtr 转为 *bool；receiver 为 nil 时返回 nil（用于 omitempty 透传上游）。
func (b *BoolValue) BoolPtr() *bool {
	if b == nil {
		return nil
	}
	v := bool(*b)
	return &v
}
