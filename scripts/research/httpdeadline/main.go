// Standard-library feasibility diagnostic, not a Grace endpoint integration test.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ordinary", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/extended", func(w http.ResponseWriter, r *http.Request) {
		if http.NewResponseController(w).SetWriteDeadline(time.Now().Add(300*time.Millisecond)) != nil {
			panic("deadline unsupported")
		}
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	})
	server := httptest.NewUnstartedServer(mux)
	server.Config.WriteTimeout = 20 * time.Millisecond
	server.Start()
	defer server.Close()
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: time.Second}
	fetch := func(path string) bool {
		resp, err := client.Get(server.URL + path)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == 200
	}
	defaultFailed := !fetch("/ordinary")
	extendedPassed := fetch("/extended")
	otherStillFails := !fetch("/ordinary")
	if !defaultFailed || !extendedPassed || !otherStillFails {
		panic("HTTP deadline diagnostic disagrees with contract")
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"diagnostic": "standard-library HTTP server, not Grace route", "defaultSlowFailed": defaultFailed, "routeDeadlinePassed": extendedPassed, "otherRouteStillFails": otherStillFails, "providerCalls": 0, "databaseWrites": 0})
}
