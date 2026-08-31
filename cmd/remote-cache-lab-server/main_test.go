package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheRoundTripAndManifest(t *testing.T) {
	state := &cacheState{root: t.TempDir()}
	put := httptest.NewRequest(http.MethodPut, "/cache/key", bytes.NewBufferString("payload"))
	putResponse := httptest.NewRecorder()
	state.cache(putResponse, put)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT status = %d", putResponse.Code)
	}
	getResponse := httptest.NewRecorder()
	state.cache(getResponse, httptest.NewRequest(http.MethodGet, "/cache/key", nil))
	if getResponse.Code != http.StatusOK || getResponse.Body.String() != "payload" {
		t.Fatalf("GET = %d/%q", getResponse.Code, getResponse.Body.String())
	}
	manifestResponse := httptest.NewRecorder()
	state.manifest(manifestResponse, httptest.NewRequest(http.MethodGet, "/manifest", nil))
	var manifest struct {
		ObjectCount int   `json:"objectCount"`
		ObjectBytes int64 `json:"objectBytes"`
		Puts        int64 `json:"puts"`
		GetHits     int64 `json:"getHits"`
	}
	if err := json.Unmarshal(manifestResponse.Body.Bytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ObjectCount != 1 || manifest.ObjectBytes != 7 || manifest.Puts != 1 || manifest.GetHits != 1 {
		t.Fatalf("manifest = %+v", manifest)
	}
}
