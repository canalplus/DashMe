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
	serverChan := make(chan error)
	/* Adding /manifest route */
	s.addRoute("GET", "/files", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		res, err := json.Marshal(cache.GetAvailables())
		fmt.Fprintf(w, string(res))
		if err != nil {
			serverChan <- err
		}
	})
	/* Adding /mem route */
	s.addRoute("GET", "/mem", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		utils.DisplayMemStats()
		fmt.Fprintf(w, "")
	})
	/* Adding /manifest/<filename> route */
	s.addRoute("GET", "/dash/:filename/manifest.mpd", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetManifest(params["filename"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	})
	/* Adding /dash/<filename>/<chunk> route */
	s.addRoute("GET", "/dash/:filename/:chunk", func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetChunk(params["filename"], params["chunk"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	})
	/* Start file monitoring */
	inotifyChan, err := StartInotify(&cache, *videoDir)
	if err != nil {
		fmt.Printf("Failed to initialise INOTIFY\n")
	}
	/* Starting API */
	fmt.Printf("GO Version : " + runtime.Version() + "\n")
	fmt.Printf("Starting DashMe API (video=%q, cache=%q), listening on port %q\n", *videoDir, *cachedDir, *port)
	go s.start(*port)
	for {
		select {
		case serverError := <- serverChan:
			fmt.Printf("Server Error : %q\n", serverError.Error())
		case inotifyError := <- inotifyChan:
			fmt.Printf("Inotify Error : %q\n", inotifyError.Error())
		}
	}
}
