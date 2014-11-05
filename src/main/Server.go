package main

import (
	"fmt"
	"utils"
	"net/http"
)

/* Function type use to hanlde a http request */
type RouteHandler func(w http.ResponseWriter, r *http.Request, params map[string]string)

/* Structure used to represent a http route */
type Route struct {
	handler RouteHandler
	pattern string
	method  string
}

/* Structure used to store server specific information */
type Server struct {
	routes []Route
}

/* Add a route to a server */
func (s *Server) addRoute(method string, pattern string, handler RouteHandler) {
	s.routes = append(s.routes, Route{handler : handler, pattern : pattern, method : method})
}

/* Remove a route from a server */
func (s *Server) removeRoute(method string, pattern string) {
	for i := 0; i < len(s.routes); i++ {
		if s.routes[i].method == method && s.routes[i].pattern == pattern {
			s.routes = append(s.routes[:i], s.routes[i+1:]...)
			break
		}
	}
}

/* Find correct route and return its handler */
func (s *Server) getRouteHandler(method string, path string, params *map[string]string) (RouteHandler, int) {
	var i int
	/* Find corresponding route */
	for i = 0; i < len(s.routes); i++ {
		if utils.ParseURL(s.routes[i].pattern, path, params) {
			break
		}
	}
	/* Not found : return 404 */
	if i == len(s.routes) {
		return nil, http.StatusNotFound
	}
	/* Found but the method used is not defined : return 405 */
	if s.routes[i].method != method {
		return nil, http.StatusMethodNotAllowed
	}
	/* Return its handler */
	return s.routes[i].handler, 0
}

/* Start sever */
func (s *Server) start(port string) {
	/* Set global handler for any request */
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var status int
		params := make(map[string]string)
		/* Get handler corresponding to route call */
		handler, status := s.getRouteHandler(r.Method, r.URL.Path, &params)
		fmt.Printf("[" + r.Method + "] " + r.URL.Path + "\n")
		/* If we have an handler, call it, otherwise return error code */
		if handler != nil {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "GET")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
			handler(w, r, params)
		} else {
			fmt.Printf("Error while serving : No handler found for route !\n")
			http.Error(w, "Invalid request !", status)
		}
	})
	/* start listening on provided port */
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		fmt.Printf("Error while server listening : ", err)
	}
}
