package main

import (
	"fmt"
	"flag"
	"runtime"
	"net/http"
	"encoding/json"
)

const (
	DEFAULT_PORT       = "3000"
	DEFAULT_VIDEO_DIR  = "/home/aubin/Workspace/videos/"
	DEFAULT_CACHED_DIR = "/tmp/DashMe"
)

/* /files handler */
func filesRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		res, err := json.Marshal(cache.GetAvailables())
		fmt.Fprintf(w, string(res))
		if err != nil {
			serverChan <- err
		}
	}
}

/* /files handler */
func filesAddRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		var av available
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&av)
		if err == nil {
			err = cache.AddAvailable(av)
		}
		if err != nil {
			http.Error(w, "Invalid request !", http.StatusBadRequest)
			serverChan <- err
		} else {
			fmt.Fprintf(w, "")
		}
	}
}

/* /manifest/<filename> handler */
func manifestRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetManifest(params["filename"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	}
}

/* /manifest/<filename>/<chunk> handler */
func chunkRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetChunk(params["filename"], params["chunk"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	}
}

func parseCommandLine(port *string, videoDir *string, cachedDir *string) {
	tmpPort := flag.String("port", DEFAULT_PORT, "TCP port used when starting the API")
	tmpVideoDir := flag.String("video", DEFAULT_VIDEO_DIR, "Directory containing the videos")
	tmpCachedDir := flag.String("cache", DEFAULT_CACHED_DIR, "Directory used for caching")
	flag.Parse()
	if *tmpPort == "" {
		*port = DEFAULT_PORT
	} else {
		*port = *tmpPort
	}
	if *tmpVideoDir == "" {
		*videoDir = DEFAULT_VIDEO_DIR
	} else {
		*videoDir = *tmpVideoDir
	}
	if *tmpCachedDir == "" {
		*cachedDir = DEFAULT_CACHED_DIR
	} else {
		*cachedDir = *tmpCachedDir
	}
}


/* Main function */
func main() {
	var server    Server
	var cache     CacheManager
	var logger    Logger
	var port      string
	var videoDir  string
	var cachedDir string
	/* Parsing command line */
	parseCommandLine(&port, &videoDir, &cachedDir)
	/* Initialising data structures */
	cache.Initialise(videoDir, cachedDir)
	serverChan := make(chan error)
	/* Initialise route handling */
	server.addRoute("GET", "/files", filesRouteHandler(&cache, serverChan))
	server.addRoute("POST", "/files", filesAddRouteHandler(&cache, serverChan))
	server.addRoute("GET", "/dash/:filename/manifest.mpd", manifestRouteHandler(&cache, serverChan))
	server.addRoute("GET", "/dash/:filename/:chunk", chunkRouteHandler(&cache, serverChan))
	/* Start file monitoring */
	inotifyChan, err := StartInotify(&cache, videoDir)
	if err != nil {
		logger.Error("Failed to initialise INOTIFY")
	}
	/* Starting API */
	logger.Debug("GO Version : " + runtime.Version())
	logger.Debug("Starting DashMe API (video=%q, cache=%q), listening on port %q", videoDir, cachedDir, port)
	go server.start(port, serverChan, logger)
	/* Wait for error from both Inotify and Serve threads */
	for {
		select {
		case serverError := <- serverChan:
			logger.Error("Server Error : %q", serverError.Error())
		case inotifyError := <- inotifyChan:
			logger.Error("Inotify Error : %q", inotifyError.Error())
		}
	}
}
