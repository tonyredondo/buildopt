// Command remote-cache-lab-server provides a loopback-only Gradle HTTP Build
// Cache origin for bounded correctness experiments. It is not a product server.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type cacheEntry struct {
	Key    string `json:"key"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type cacheState struct {
	root string
	mu   sync.Mutex
	puts int64
	gets int64
}

func main() {
	listen := flag.String("listen", "127.0.0.1:0", "loopback listen address")
	root := flag.String("root", "", "object directory")
	flag.Parse()
	if *root == "" {
		log.Fatal("--root is required")
	}
	if err := os.MkdirAll(*root, 0o700); err != nil {
		log.Fatal(err)
	}
	state := &cacheState{root: *root}
	mux := http.NewServeMux()
	mux.HandleFunc("/cache/", state.cache)
	mux.HandleFunc("/manifest", state.manifest)
	mux.HandleFunc("/reset-stats", state.reset)
	server := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 0}
	fmt.Printf("http://%s\n", *listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (state *cacheState) cache(response http.ResponseWriter, request *http.Request) {
	key := strings.TrimPrefix(request.URL.Path, "/cache/")
	if key == "" || strings.ContainsAny(key, "/\\") {
		http.Error(response, "invalid key", http.StatusBadRequest)
		return
	}
	path := filepath.Join(state.root, key)
	switch request.Method {
	case http.MethodPut:
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(response, "read failed", http.StatusBadRequest)
			return
		}
		temporary, err := os.CreateTemp(state.root, ".put-*")
		if err != nil {
			http.Error(response, "store failed", http.StatusInternalServerError)
			return
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		if _, err = temporary.Write(payload); err == nil {
			err = temporary.Sync()
		}
		if closeErr := temporary.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(temporaryPath, path)
		}
		if err != nil {
			http.Error(response, "store failed", http.StatusInternalServerError)
			return
		}
		state.mu.Lock()
		state.puts++
		state.mu.Unlock()
		response.WriteHeader(http.StatusOK)
	case http.MethodGet:
		payload, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(response, "read failed", http.StatusInternalServerError)
			return
		}
		state.mu.Lock()
		state.gets++
		state.mu.Unlock()
		response.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = response.Write(payload)
	default:
		response.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (state *cacheState) manifest(response http.ResponseWriter, _ *http.Request) {
	directory, err := os.ReadDir(state.root)
	if err != nil {
		http.Error(response, "manifest failed", http.StatusInternalServerError)
		return
	}
	entries := make([]cacheEntry, 0, len(directory))
	var bytes int64
	for _, item := range directory {
		if item.IsDir() || strings.HasPrefix(item.Name(), ".put-") {
			continue
		}
		payload, readErr := os.ReadFile(filepath.Join(state.root, item.Name()))
		if readErr != nil {
			http.Error(response, "manifest failed", http.StatusInternalServerError)
			return
		}
		digest := sha256.Sum256(payload)
		entries = append(entries, cacheEntry{Key: item.Name(), SHA256: hex.EncodeToString(digest[:]), Bytes: int64(len(payload))})
		bytes += int64(len(payload))
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Key < entries[right].Key })
	state.mu.Lock()
	puts, gets := state.puts, state.gets
	state.mu.Unlock()
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{"entries": entries, "objectCount": len(entries), "objectBytes": bytes, "puts": puts, "getHits": gets})
}

func (state *cacheState) reset(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	state.mu.Lock()
	state.puts, state.gets = 0, 0
	state.mu.Unlock()
	response.WriteHeader(http.StatusNoContent)
}
