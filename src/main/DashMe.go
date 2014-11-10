package main

import (
	"fmt"
	"flag"
	"utils"
	"runtime"
	"net/http"
	"encoding/json"
)

const (
	DEFAULT_PORT       = "3000"
	DEFAULT_VIDEO_DIR  = "/home/aubin/Workspace/videos/"
	DEFAULT_CACHED_DIR = "/tmp/DashMe"
)

func main() {
	var s Server
	var cache CacheManager
	/* Parsing command line */
	port  := flag.String("port", DEFAULT_PORT, "TCP port used when starting the API")
	videoDir := flag.String("video", DEFAULT_VIDEO_DIR, "Directory containing the videos")
	cachedDir := flag.String("cache", DEFAULT_CACHED_DIR, "Directory used for caching")
	flag.Parse()
	if *port == "" { *port = DEFAULT_PORT }
	if *videoDir == "" { *videoDir = DEFAULT_VIDEO_DIR }
	if *cachedDir == "" { *cachedDir = DEFAULT_CACHED_DIR }
	/* Initialising data structures */
	cache.Initialise(*videoDir, *cachedDir)
	/* Adding /manifest route */
	s.addRoute("GET", "/files", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		res, _ := json.Marshal(cache.GetAvailables())
		fmt.Fprintf(w, string(res))
	})
	/* Adding /manifest route */
	s.addRoute("GET", "/mem", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		utils.DisplayMemStats()
		fmt.Fprintf(w, "")
	})
	/* Adding /manifest/<filename> route */
	s.addRoute("GET", "/dash/:filename/manifest.mpd", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetManifest(params["filename"])
		if err != nil {
			fmt.Printf("Error while retrieving manifest : " + err.Error() + "\n")
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	})
	/* Adding /dash/<filename>/<chunk> route */
	s.addRoute("GET", "/dash/:filename/:chunk", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetChunk(params["filename"], params["chunk"])
		if err != nil {
			fmt.Printf("Error while retrieving chunk : " + err.Error() + "\n")
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	})
	/* Starting API */
	fmt.Printf("GO Version : " + runtime.Version() + "\n")
	fmt.Printf("Starting DashMe API (video=%q, cache=%q), listening on port %q\n", *videoDir, *cachedDir, *port)
	s.start(*port)
}
