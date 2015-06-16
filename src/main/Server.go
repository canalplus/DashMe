// Copyright 2015 CANAL+ Group
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"utils"
	"errors"
	"strings"
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
		if utils.ParseURL(s.routes[i].pattern, path, params) && s.routes[i].method == method  {
			break
		}
	}
	/* Not found : return 404 */
	if i == len(s.routes) {
		return nil, http.StatusNotFound
	}
	/* Return its handler */
	return s.routes[i].handler, 0
}

/* Set headers for CORS */
func (s *Server) setCORSHeaders(w http.ResponseWriter, path string) {
	var i int
	var methods []string
	/* Find corresponding routes */
	for i = 0; i < len(s.routes); i++ {
		if utils.ParseURL(s.routes[i].pattern, path, nil) {
			methods = append(methods, s.routes[i].method)
		}
	}
	if len(methods) > 0 {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	}
}

/* Start sever */
func (s *Server) start(port string, errChan chan error, logger Logger) {
	/* Set global handler for any request */
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var status int
		params := make(map[string]string)
		/* Get handler corresponding to route call */
		handler, status := s.getRouteHandler(r.Method, r.URL.Path, &params)
		/* If we have an handler, call it, otherwise return error code */
		if handler != nil {
			s.setCORSHeaders(w, r.URL.Path)
			handler(w, r, params)
			logger.Debug("" + r.Method + " : " + r.URL.Path)
		} else {
			errChan <- errors.New("Unable to serve '" + r.URL.Path + "', no handler has been found")
			http.Error(w, "Invalid request !", status)
			logger.Debug("" + r.Method + " : " + r.URL.Path)
		}
	})
	/* start listening on provided port */
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		errChan <- err
	}
}
