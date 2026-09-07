package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"stackbyte-vm/internal/config"
)

func TestRunAndStaticWeb(t *testing.T) {
	server, err := NewServer(config.Default(), slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	if err != nil {
		t.Fatal(err)
	}
	webRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	webResponse := httptest.NewRecorder()
	server.Handler.ServeHTTP(webResponse, webRequest)
	if webResponse.Code != http.StatusOK || !bytes.Contains(webResponse.Body.Bytes(), []byte("StackByte VM")) {
		t.Fatalf("unexpected web response: %d", webResponse.Code)
	}

	body, _ := json.Marshal(sourceRequest{Source: "print 2 + 3;"})
	runRequest := httptest.NewRequest(http.MethodPost, "/api/v1/run", bytes.NewReader(body))
	runResponse := httptest.NewRecorder()
	server.Handler.ServeHTTP(runResponse, runRequest)
	if runResponse.Code != http.StatusOK || !bytes.Contains(runResponse.Body.Bytes(), []byte(`"output":"5\n"`)) {
		t.Fatalf("unexpected run response: %d %s", runResponse.Code, runResponse.Body.String())
	}
}
