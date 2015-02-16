package main

import (
	"os"
	"io"
	"fmt"
	"flag"
	"runtime"
	"net/http"
	"path/filepath"
	"encoding/json"
)

const (
	DEFAULT_PORT       = "3000"
	DEFAULT_VIDEO_DIR  = "/home/aubin/Workspace/videos/"
	DEFAULT_CACHED_DIR = "/tmp/DashMe"
	DEFAULT_INTERFACE_DIR  = "/home/aubin/Workspace/DashMe/interface"
)

/* GET /files handler */
func filesRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		var err error;
		str := "[]";
		w.Header().Set("Content-Type", "application/json")
		availables := cache.GetAvailables()
		if availables != nil {
			if res, err := json.Marshal(availables); err == nil {
				str = string(res)
			}
		}
		fmt.Fprintf(w, str)
		if err != nil {
			serverChan <- err
		}
	}
}

/* POST /files handler */
func filesAddRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		var av Available
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

/* POST /files/upload handler */
func filesUploadHandler(cache *CacheManager, serverChan chan error, videoDir string) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		file, header, err := r.FormFile("video")
		if err != nil {
			http.Error(w, "Invalid request !", http.StatusBadRequest)
			serverChan <- err
			return
		}
		defer file.Close()
		path := filepath.Join(videoDir, header.Filename)
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			goto exit
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			goto exit
		}
	exit:
		if err != nil {
			http.Error(w, "Invalid request !", http.StatusBadRequest)
			serverChan <- err
		} else {
			fmt.Fprintf(w, "")
		}
	}
}

/* GET /dash/<filename>/<elm> handler */
func elementRouteHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path, err := cache.GetElement(params["filename"], params["elm"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			http.ServeFile(w, r, path)
		}
	}
}

/* POST /dash/<filename>/generate handler */
func generationHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		err := cache.Build(params["filename"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			fmt.Fprintf(w, "")
		}
	}
}

/* DELETE /dash/<filename>/generate handler */
func liveStopHandler(cache *CacheManager, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		err := cache.Stop(params["filename"])
		if err != nil {
			serverChan <- err
			http.Error(w, "Invalid request !", http.StatusNotFound)
		} else {
			fmt.Fprintf(w, "")
		}
	}
}

/* GET /* */
func interfaceHandler(interfaceDir string, serverChan chan error) RouteHandler {
	return func (w http.ResponseWriter, r *http.Request, params map[string]string) {
		path := params["path"]
		if path == "" {
			path = "/index.html"
		}
		http.ServeFile(w, r, filepath.Join(interfaceDir, path))
	}
}

func parseCommandLine(port *string, videoDir *string, cachedDir *string, interfaceDir *string) {
	tmpPort := flag.String("port", DEFAULT_PORT, "TCP port used when starting the API")
	tmpVideoDir := flag.String("video", DEFAULT_VIDEO_DIR, "Directory containing the videos")
	tmpCachedDir := flag.String("cache", DEFAULT_CACHED_DIR, "Directory used for caching")
	tmpInterfaceDir := flag.String("ui", DEFAULT_INTERFACE_DIR, "Directory containing the UI")
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
	if *tmpInterfaceDir == "" {
		*interfaceDir = DEFAULT_CACHED_DIR
	} else {
		*interfaceDir = *tmpInterfaceDir
	}
}

/* Main function */
func main() {
	var server       Server
	var cache        CacheManager
	var logger       Logger
	var port         string
	var videoDir     string
	var cachedDir    string
	var interfaceDir string
	/* Parsing command line */
	parseCommandLine(&port, &videoDir, &cachedDir, &interfaceDir)
	/* Initialising data structures */
	cache.Initialise(videoDir, cachedDir)
	serverChan := make(chan error)
	/* Initialise route handling */
	server.addRoute("GET", "/files", filesRouteHandler(&cache, serverChan))
	server.addRoute("POST", "/files", filesAddRouteHandler(&cache, serverChan))
	server.addRoute("POST", "/files/upload", filesUploadHandler(&cache, serverChan, videoDir))
	server.addRoute("GET", "/dash/:filename/:elm", elementRouteHandler(&cache, serverChan))
	server.addRoute("POST", "/dash/:filename/generate", generationHandler(&cache, serverChan))
	server.addRoute("DELETE", "/dash/:filename/generate", liveStopHandler(&cache, serverChan))
	server.addRoute("GET", "/*path", interfaceHandler(interfaceDir, serverChan))
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
