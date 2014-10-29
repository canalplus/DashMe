package main

import (
	"fmt"
	"net/http"
)

type RouteHandler func(w http.ResponseWriter, r *http.Request, params map[string]string)

type Route struct {
	handler RouteHandler
	pattern string
	method  string
}

type Server struct {
	routes []Route
}

func (s *Server) addRoute(method string, pattern string, handler RouteHandler) {
	s.routes = append(s.routes, Route{handler : handler, pattern : pattern, method : method})
}

func (s *Server) removeRoute(method string, pattern string) {
	for i := 0; i < len(s.routes); i++ {
		if s.routes[i].method == method && s.routes[i].pattern == pattern {
			s.routes = append(s.routes[:i], s.routes[i+1:]...)
			break
		}
	}
}

func (s *Server) getRouteHandler(method string, path string, params *map[string]string) (RouteHandler, int) {
	var i int
	for i = 0; i < len(s.routes); i++ {
		if parseURL(s.routes[i].pattern, path, params) {
			break
		}
	}
	if i == len(s.routes) {
		return nil, http.StatusNotFound
	}
	if s.routes[i].method != method {
		return nil, http.StatusMethodNotAllowed
	}
	return s.routes[i].handler, 0
}

func (s *Server) start(port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var status int
		params := make(map[string]string)
		handler, status := s.getRouteHandler(r.Method, r.URL.Path, &params)
		fmt.Printf("[" + r.Method + "] " + r.URL.Path + "\n")
		if handler != nil {
			handler(w, r, params)
		} else {
			fmt.Printf("Error while serving : No handler found for route !\n")
			http.Error(w, "Invalid request !", status)
		}
	})
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		fmt.Printf("Error while server listening : ", err)
	}
}
