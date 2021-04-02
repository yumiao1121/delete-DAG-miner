package model

import "encoding/json"

type JSONRpcResp struct {
	Id      json.RawMessage `json:"id"`
	Version string          `json:"version"`
	Result  interface{}     `json:"result"`
	Error   interface{}     `json:"error"`
}

type JSONRpcReq struct {
	Id     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}
