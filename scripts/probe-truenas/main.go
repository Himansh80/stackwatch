package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

func main() {
	url := "wss://192.168.0.112/api/current"
	password := "@himanshutruenasadmin&806376"

	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	c, _, err := dialer.Dial(url, nil)
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	defer c.Close()

	call := func(method string, params []any) map[string]any {
		c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": method, "method": method, "params": params})
		var resp map[string]any
		c.ReadJSON(&resp)
		return resp
	}

	if r := call("auth.login", []any{"truenas_admin", password}); r["result"] != true {
		fmt.Println("login failed")
		return
	}

	test := func(method, label string, params []any) {
		r := call(method, params)
		bs, _ := json.Marshal(r)
		out, _ := r["result"].([]any)
		fmt.Printf("%-25s  %-22s  result_len=%d  raw=%s\n", method, label, len(out), truncate(string(bs), 200))
	}

	test("pool.query", "[]", []any{})
	test("pool.query", "[[]]", []any{[][]any{}})
	test("pool.query", "[null]", []any{nil})
	test("pool.query", "[[],{}]", []any{[][]any{}, map[string]any{}})
	test("pool.query", "[null,{}]", []any{nil, map[string]any{}})

	test("pool.dataset.query", "[]", []any{})
	test("pool.dataset.query", "[[]]", []any{[][]any{}})
	test("pool.dataset.query", "[null]", []any{nil})
	test("pool.dataset.query", "[null,{}]", []any{nil, map[string]any{}})

	test("user.query", "[null,{}]", []any{nil, map[string]any{"limit": 2}})
	test("service.query", "[null,{}]", []any{nil, map[string]any{}})
	test("sharing.nfs.query", "[[]]", []any{[][]any{}})
	test("disk.query", "[[]]", []any{[][]any{}})
	test("boot.environment.query", "[[]]", []any{[][]any{}})
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
