package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type apiClient struct {
	baseURL    string
	host       string
	httpClient *http.Client
	localToken string
	jwtToken   string
}

func newAPIClient(host string, port int) *apiClient {
	return &apiClient{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		host:    host,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		localToken: "Hamburger",
	}
}

func (c *apiClient) doJSON(method string, path string, payload interface{}, auth bool) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Hamburger-Token", c.localToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && strings.TrimSpace(c.jwtToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.jwtToken))
	}
	if c.host == "127.0.0.1" {
		req.Host = "127.0.0.1"
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func (c *apiClient) requestAndPrint(method string, path string, payload interface{}, auth bool) error {
	body, code, err := c.doJSON(method, path, payload, auth)
	if err != nil {
		return err
	}
	printResponse(code, body)
	return nil
}

func (c *apiClient) handleToken(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: token show|set <jwt>|clear")
	}
	switch strings.ToLower(args[1]) {
	case "show":
		if strings.TrimSpace(c.jwtToken) == "" {
			fmt.Println("jwt token: <empty>")
			return nil
		}
		fmt.Println("jwt token:", c.jwtToken)
		return nil
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: token set <jwt>")
		}
		c.jwtToken = strings.TrimSpace(args[2])
		fmt.Println(commandText("token"), "updated")
		return nil
	case "clear":
		c.jwtToken = ""
		fmt.Println(commandText("token"), "cleared")
		return nil
	default:
		return fmt.Errorf("usage: token show|set <jwt>|clear")
	}
}

func (c *apiClient) handleLogin(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: login <username> <password>")
	}
	payload := map[string]string{
		"username": args[1],
		"password": args[2],
	}
	body, code, err := c.doJSON(http.MethodPost, "/api/login", payload, false)
	if err != nil {
		return err
	}
	printResponse(code, body)
	if code != http.StatusOK {
		return nil
	}
	var parsed map[string]interface{}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	if token, ok := parsed["token"].(string); ok && strings.TrimSpace(token) != "" {
		c.jwtToken = token
		fmt.Println(commandText("login"), "token captured")
	}
	return nil
}

func (c *apiClient) handleUser(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: user get|create|update|delete ...")
	}
	switch strings.ToLower(args[1]) {
	case "get":
		return c.requestAndPrint(http.MethodGet, "/api/user", nil, true)
	case "create":
		if len(args) < 4 {
			return fmt.Errorf("usage: user create <username> <password> [nickname] [avatar]")
		}
		payload := map[string]string{
			"username": args[2],
			"password": args[3],
			"nickname": valueOrEmpty(args, 4),
			"avatar":   valueOrEmpty(args, 5),
		}
		return c.requestAndPrint(http.MethodPost, "/api/user", payload, true)
	case "update":
		if len(args) < 4 {
			return fmt.Errorf("usage: user update <username> <password> [nickname] [avatar]")
		}
		payload := map[string]string{
			"username": args[2],
			"password": args[3],
			"nickname": valueOrEmpty(args, 4),
			"avatar":   valueOrEmpty(args, 5),
		}
		return c.requestAndPrint(http.MethodPut, "/api/user", payload, true)
	case "delete":
		payload := map[string]string{
			"username": valueOrEmpty(args, 2),
		}
		return c.requestAndPrint(http.MethodDelete, "/api/user", payload, true)
	default:
		return fmt.Errorf("usage: user get|create|update|delete ...")
	}
}

func (c *apiClient) handleService(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: service start|stop <domain>")
	}
	action := strings.ToLower(args[1])
	payload := map[string]string{
		"domain": strings.TrimSpace(args[2]),
	}
	switch action {
	case "start":
		return c.requestAndPrint(http.MethodPost, "/api/service/start", payload, true)
	case "stop":
		return c.requestAndPrint(http.MethodPost, "/api/service/stop", payload, true)
	default:
		return fmt.Errorf("usage: service start|stop <domain>")
	}
}

func (c *apiClient) handleServer(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: server stop|restart <gateway|front|api|backend|latency|vpn|trojan|anytls>")
	}
	action := strings.ToLower(args[1])
	payload := map[string]string{
		"server": strings.TrimSpace(args[2]),
	}
	switch action {
	case "stop":
		return c.requestAndPrint(http.MethodPost, "/api/server/stop", payload, true)
	case "restart":
		return c.requestAndPrint(http.MethodPost, "/api/server/restart", payload, true)
	default:
		return fmt.Errorf("usage: server stop|restart <gateway|front|api|backend|latency|vpn|trojan|anytls>")
	}
}

func valueOrEmpty(args []string, index int) string {
	if index >= len(args) {
		return ""
	}
	return strings.TrimSpace(args[index])
}

func printResponse(status int, body []byte) {
	fmt.Printf("HTTP %d\n", status)
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		fmt.Println("{}")
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, trimmed, "", "  "); err != nil {
		fmt.Println(string(trimmed))
		return
	}
	fmt.Println(pretty.String())
}

