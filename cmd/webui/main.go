package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// Simple Web UI + API starter.
// - Serves static HTML/CSS/JS
// - Accepts file uploads
// - Runs a tiny "processing" step (hash/size/preview)
// - Returns results for display in the browser
//
// This is intentionally minimal: no DB, no auth, in-memory storage.
// For production: add auth, store uploads/results in GCS/Firestore/Cloud SQL, add worker queue if needed.

//go:embed Web/*
var webFS embed.FS

type Job struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`

	Status   string `json:"status"` // queued|done|error
	Error    string `json:"error,omitempty"`
	Checksum string `json:"checksum,omitempty"`
	Preview  string `json:"preview,omitempty"` // first N bytes as text (best-effort)
}

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Put(j *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.ID] = j
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	// Small: not sorting to keep code minimal
	return out
}

func main() {
	store := NewStore()

	mux := http.NewServeMux()

	// Static UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for root
		if r.URL.Path == "/" {
			serveEmbeddedFile(w, r, "web/index.html")
			return
		}
		// Serve assets under /static
		if strings.HasPrefix(r.URL.Path, "/static/") {
			serveEmbeddedFile(w, r, path.Join("web", r.URL.Path))
			return
		}
		http.NotFound(w, r)
	})

	// API: upload file
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit body size (10MB) to prevent accidental huge uploads
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "bad multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field named 'file': "+err.Error(), http.StatusBadRequest)
			return
		}
		defer f.Close()

		now := time.Now()
		jobID := fmt.Sprintf("%d-%08x", now.UnixNano(), crc32.ChecksumIEEE([]byte(hdr.Filename+now.String())))
		j := &Job{
			ID:        jobID,
			Filename:  hdr.Filename,
			CreatedAt: now,
			Status:    "queued",
		}
		store.Put(j)

		// Process synchronously (simple MVP). For heavier work, queue it.
		if err := processFile(j, f); err != nil {
			j.Status = "error"
			j.Error = err.Error()
		} else {
			j.Status = "done"
		}

		writeJSON(w, http.StatusOK, map[string]string{"id": jobID})
	})

	// API: list jobs
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, store.List())
	})

	// API: get job by id
	mux.HandleFunc("/api/job/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/job/")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		j, ok := store.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, j)
	})

	addr := ":8080"
	fmt.Println("Web UI running on http://localhost" + addr)
	if err := http.ListenAndServe(addr, withBasicHeaders(mux)); err != nil {
		panic(err)
	}
}

func processFile(j *Job, r io.Reader) error {
	// Compute size + checksum, and capture a small preview for UI display
	const previewMax = 4096

	var size int64
	hasher := crc32.NewIEEE()
	previewBuf := make([]byte, 0, previewMax)
	tmp := make([]byte, 32*1024)

	for {
		n, err := r.Read(tmp)
		if n > 0 {
			size += int64(n)
			_, _ = hasher.Write(tmp[:n])

			remain := previewMax - len(previewBuf)
			if remain > 0 {
				if n < remain {
					previewBuf = append(previewBuf, tmp[:n]...)
				} else {
					previewBuf = append(previewBuf, tmp[:remain]...)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	j.SizeBytes = size
	j.Checksum = fmt.Sprintf("%08x", hasher.Sum32())

	// Best-effort preview as text
	j.Preview = string(previewBuf)
	return nil
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, name string) {
	b, err := webFS.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Basic content-type setting
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func withBasicHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal security headers (good defaults)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
