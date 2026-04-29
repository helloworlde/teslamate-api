package main

import "encoding/json"

// TeslaUpstreamJSON Tesla Fleet API / TeslaMate 日志接口返回的原始 JSON 字节。
// OpenAPI 中视为不透明 JSON；运行时不解析为 map。
type TeslaUpstreamJSON json.RawMessage

// MarshalJSON 原样输出字节。
func (t TeslaUpstreamJSON) MarshalJSON() ([]byte, error) {
	if len(t) == 0 {
		return []byte("{}"), nil
	}
	return []byte(t), nil
}
