package lua

import (
	"net/http"
)

func (m *manager) handleRequest(request *http.Request) error {
	if !m.enabled || len(m.middlewares) == 0 {
		return nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	ensureURL(request)
	for _, item := range m.middlewares {
		reqTable := buildRequestTable(m.state, request)
		if err := m.state.CallByParam(luaParam(item.function), reqTable); err != nil {
			return err
		}
		result, err := parseRequestResult(m.state.Get(-1))
		m.state.Pop(1)
		if err != nil {
			return err
		}
		applyRequestResult(request, result)
		if result != nil && ((result.Allow != nil && !*result.Allow) || result.Error != "") {
			return rejectError(result.Error)
		}
	}
	return nil
}
