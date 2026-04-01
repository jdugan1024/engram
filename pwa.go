// ABOUTME: PWA support — manifest, service worker, and icon serving.

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"net/http"
	"strconv"
	"sync"

	xdraw "golang.org/x/image/draw"
)

//go:embed web/icon.png
var iconSrc []byte

// Resized icon cache — generated once per size on first request.
var (
	iconCache   = map[int][]byte{}
	iconCacheMu sync.Mutex
)

func iconHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sizeStr := r.URL.Query().Get("s")
		size, _ := strconv.Atoi(sizeStr)
		if size <= 0 || size > 512 {
			size = 192
		}

		iconCacheMu.Lock()
		cached, ok := iconCache[size]
		iconCacheMu.Unlock()

		if !ok {
			src, err := png.Decode(bytes.NewReader(iconSrc))
			if err != nil {
				http.Error(w, "icon decode error", http.StatusInternalServerError)
				return
			}
			dst := image.NewRGBA(image.Rect(0, 0, size, size))
			xdraw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

			var buf bytes.Buffer
			png.Encode(&buf, dst)
			cached = buf.Bytes()

			iconCacheMu.Lock()
			iconCache[size] = cached
			iconCacheMu.Unlock()
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(cached)
	}
}

var manifestJSON = `{
  "name": "Engram",
  "short_name": "Engram",
  "description": "Personal knowledge capture",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#111827",
  "theme_color": "#4f46e5",
  "icons": [
    { "src": "/icon?s=192", "sizes": "192x192", "type": "image/png", "purpose": "any maskable" },
    { "src": "/icon?s=512", "sizes": "512x512", "type": "image/png", "purpose": "any maskable" }
  ]
}
`

var serviceWorkerJS = `
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', e => e.waitUntil(clients.claim()));
`

func RegisterPWAHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(manifestJSON))
	})
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(serviceWorkerJS))
	})
	mux.HandleFunc("GET /icon", iconHandler())
}
