package main

import (
	"encoding/json"
	"strings"
	"unsafe"
)

var buffers = map[uint32][]byte{}

//export alloc
func alloc(size uint32) uint32 {
	if size == 0 {
		size = 1
	}
	buf := make([]byte, size)
	ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))
	buffers[ptr] = buf
	return ptr
}

//export free
func free(ptr uint32, length uint32) {
	delete(buffers, ptr)
}

//export request_handle
func request_handle(ptr uint32, length uint32) uint64 {
	data := read(ptr, length)
	if len(data) == 0 {
		return 0
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0
	}
	method, _ := payload["method"].(string)
	headers := getHeaderMap(payload["header"])
	expectMethod := headerValue(headers, "X-Allow-Method")
	if expectMethod == "" || strings.EqualFold(method, expectMethod) {
		return 0
	}
	result := map[string]interface{}{
		"allow":  false,
		"error":  "method not allowed",
		"status": 403,
	}
	out, err := json.Marshal(result)
	if err != nil {
		return 0
	}
	outPtr := alloc(uint32(len(out)))
	write(outPtr, out)
	return pack(outPtr, uint32(len(out)))
}

func read(ptr uint32, length uint32) []byte {
	if length == 0 {
		return nil
	}
	mem := (*[1 << 30]byte)(unsafe.Pointer(uintptr(ptr)))[:length:length]
	out := make([]byte, length)
	copy(out, mem)
	return out
}

func write(ptr uint32, data []byte) {
	if len(data) == 0 {
		return
	}
	mem := (*[1 << 30]byte)(unsafe.Pointer(uintptr(ptr)))[:len(data):len(data)]
	copy(mem, data)
}

func pack(ptr uint32, length uint32) uint64 {
	return uint64(ptr) | (uint64(length) << 32)
}

func getHeaderMap(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	if headers, ok := value.(map[string]interface{}); ok {
		return headers
	}
	return nil
}

func headerValue(headers map[string]interface{}, key string) string {
	if headers == nil {
		return ""
	}
	for k, v := range headers {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch vv := v.(type) {
		case string:
			return vv
		case []interface{}:
			if len(vv) == 0 {
				return ""
			}
			if s, ok := vv[0].(string); ok {
				return s
			}
			return ""
		}
	}
	return ""
}

func main() {}
