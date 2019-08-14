package julla

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type (

	RouteMux struct {

		urlPath string

		treeNode *Node

		context  Context

		syncPool sync.Pool

	}

)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *Context)
}

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

func (handlerFunc HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, ctx *Context) {
	handlerFunc(w, r, ctx)
}


func (mux *RouteMux)ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx := mux.NewContext(r, w)

	handler, pattern := mux.Handler(r, ctx)

	ctx.Pattern = pattern

	handler.ServeHTTP(w, r, ctx)

	mux.syncPool.Put(ctx)

}


//static resource file server handler
func (mux *RouteMux) Resources(pattern string, handler Handler) {

	if strings.HasSuffix(pattern, "/") {

		pattern = strings.Join([]string{ pattern, "?static" }, "")

	}else {

		panic("julla.Resources pattern can only end with '/'")
	}

	mux.Handle(pattern, []string{"GET"}, handler)

}

func (mux *RouteMux) Handle(pattern string, methods []string, handle Handler) {

	if pattern == "" {

		panic("julla: pattern cannot be empty")
	}

	if mux.treeNode == nil {

		mux.treeNode = &Node{

			path : "/",

			pattern: "/",

			nType : "root",

			priority : 0,

		}
	}

	mux.Enroll(pattern, methods, handle)
}

func (mux *RouteMux) HandleFunc(pattern string, methods []string, handleFunc func(w http.ResponseWriter, r *http.Request, ctx *Context)) {

	if handleFunc == nil {

		panic("http: nil handler")
	}

	mux.Handle(pattern, methods, HandlerFunc(handleFunc))

}

func (mux *RouteMux) Handler(r *http.Request, ctx *Context) (handler Handler, pattern string) {

	urlPath := ctx.UrlPath()

	handler, pattern = mux.match(urlPath, r.Method, ctx)

	return
}

func (mux *RouteMux) Enroll(pattern string, methods []string, handler Handler) {

	rex := regexp.MustCompile(`(?U).*/|.*$`)

	paths := rex.FindAllString(pattern, -1)

	node := Node{}

	if len(paths) == 1 {

		methodHandler := make(map[string]Handler)

		for _, method := range methods {

			method = strings.ToUpper(method)

			methodHandler[method] = handler
		}

		node.path = paths[0]

		node.nType = "root"

		node.priority = 0

		node.methodHandler = methodHandler

		node.pattern = pattern

		mux.enroll(node.pattern, node)

		return
	}

	for index, path := range paths {

		if index == 0 {

			continue
		}

		switch index {

		case len(paths) - 1 :

			methodHandler := make(map[string]Handler)

			for _, method := range methods {

				method = strings.ToUpper(method)

				methodHandler[method] = handler
			}

			node.path = path

			node.nType, node.alias, node.match = pathFormat(true, path)

			node.priority = index

			node.methodHandler = methodHandler

			node.pattern = strings.Join(paths[:index+1], "")

		default:

			node.nType, node.alias, node.match = pathFormat(false, path)

			node.path = path

			node.priority = index

			node.pattern = strings.Join(paths[:index+1], "")

		}

		mux.enroll(node.pattern, node)
	}


	return
}

func (mux *RouteMux) NewContext(r *http.Request, w http.ResponseWriter) *Context {

	mux.syncPool.New = func() interface{} {

		return new(Context)
	}

	ctx := mux.syncPool.Get().(*Context)

	ctx.request = r

	ctx.responseWriter = w

	return ctx

}

func (mux *RouteMux) enroll(pattern string, node Node) {

	rex := regexp.MustCompile(`(?U).*/|.*$`)

	paths := rex.FindAllString(pattern, -1)

	if len(paths) == 1 {

		mux.treeNode.methodHandler = node.methodHandler

		return
	}

	tree := mux.treeNode


	for index, _ := range paths {

		if index == 0 {

			continue
		}

		switch index {

		case len(paths) - 1 :

		    if tree.children == nil {

				tree.children = append(tree.children, &node)

				return
			}

			for i, n := range tree.children {

				if n.pattern == node.pattern {

					n.methodHandler = node.methodHandler

					return
				}

				if len(tree.children) == i + 1 {

					tree.children = append(tree.children, &node)

					return
				}

			}

		default:

			for i, n := range tree.children {

				if n.pattern == strings.Join(paths[:index+1], "") {

					tree = tree.children[i]

					continue
				}

			}

		}
	}



}

func (mux *RouteMux) match(path, method string, ctx *Context) (Handler, string){

	tree := mux.treeNode

	if path == "/" && tree.methodHandler != nil {

		return tree.methodHandler[method], "/"
	}

	index := len(mux.treeNode.children) * 5

	path = strings.TrimPrefix(path, "/")

	ch := make(chan julla, index)

	matchGroup := new(sync.WaitGroup)

	for _, child := range mux.treeNode.children {

		matchGroup.Add(1)

		go routeFind(path, method, child, nil, matchGroup, ch)

	}

	matchGroup.Wait(); close(ch)

	j := julla{
		priority: 0,
		handler: HandlerFunc(NotFound),
	}

	for c := range ch{

		if c.priority > j.priority {

			j = c
		}

		if c.priority == j.priority {

			if len(c.mimicry) < len(j.mimicry) {

				j = c
			}

		}

	}

	ctx.mimicry = j.mimicry

	if j.handler == nil {

		return HandlerFunc(NotFound), ""
	}

	return j.handler, j.pattern
}

func NewMux() (mux *RouteMux) {

	mux = new(RouteMux)

	return
}

func NotFound(w http.ResponseWriter, r *http.Request, ctx *Context) { http.Error(w, "404 page not found", http.StatusNotFound) }

func StripPrefix(prefix string, h http.Handler) Handler {

	if prefix == "" {
		
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request, ctx *Context){
			
			h.ServeHTTP(w, r)
			
		})
	}
	
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request, ctx *Context) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			h.ServeHTTP(w, r2)
		} else {
			NotFound(w, r, ctx)
		}
	})
}

func pathFormat(end bool, path string) (nType NodeType, alias, match string) {

	path = strings.TrimRight(path, "/")

	switch path[:1] {

	case ":" :
		// param
		nType = "param"
		alias = path[1:len(path)]

	case "{" :
		// regexp
		rex := regexp.MustCompile(`(?U)^{.*:|.*}$`)
		presets := rex.FindAllString(path, -1)
		nType = "regexp"
		alias = presets[0][1:len(presets[0])-1]
		match = presets[1][:len(presets[1])-1]

	case "?" :
		// static
		if end == false {

			panic("julla: router pattern '*' only at the end")
		}

		nType = "static"
		alias = path[1:]

	case "*" :
		// whole
		nType = "whole"
		alias = "whole"

	default:

		nType = "default"

	}

	return
}

func routeFind(path, method string, node *Node, mimicry map[string]string, wait *sync.WaitGroup, c chan julla) {

	mimicry = make(map[string]string)

	rex := regexp.MustCompile(`(?U).*/|.*$`)

	switch node.nType {

	case "default":

		patterns := rex.FindAllString(path, -1)

		if patterns[0] == node.path {

			path = strings.TrimPrefix(path, node.path)

			if path == "" {

				j := julla{
					priority: node.priority,
					pattern:  node.pattern,
					mimicry:  mimicry,
					handler:  node.methodHandler[method],
				}

				c <- j

			} else {

				for _, child := range node.children {

					wait.Add(1)

					go routeFind(path, method, child, mimicry, wait, c)

				}

			}

		}

	case "regexp":

		rew := regexp.MustCompile("^" + node.match + "$")

		patterns := rex.FindAllString(path, -1)

		if rew.MatchString(patterns[0]) {

			mimicry[node.alias] = patterns[0]

			path = strings.TrimPrefix(path, patterns[0])

			if path == "" {

				j := julla{
					priority: node.priority,
					pattern:  node.pattern,
					mimicry:  mimicry,
					handler:  node.methodHandler[method],
				}

				c <- j

			} else {

				for _, child := range node.children {

					wait.Add(1)

					go routeFind(path, method, child, mimicry, wait, c)

				}

			}

		}

	case "param":

		patterns := rex.FindAllString(path, -1)

		mimicry[node.alias] = patterns[0]

		path = strings.TrimPrefix(path, patterns[0])

		if path == "" {

			j := julla{
				priority: node.priority,
				pattern:  node.pattern,
				mimicry:  mimicry,
				handler:  node.methodHandler[method],
			}

			c <- j

		} else {

			for _, child := range node.children {

				wait.Add(1)

				go routeFind(path, method, child, mimicry, wait, c)

			}

		}

	case "static":

		mimicry[node.alias] = path

		j := julla{
			priority: node.priority,
			pattern:  node.pattern,
			mimicry:  mimicry,
			handler:  node.methodHandler[method],
		}

		c <- j


	case "whole":

		mimicry[node.alias] = path

		j := julla{
			priority: node.priority,
			pattern:  node.pattern,
			mimicry:  mimicry,
			handler:  node.methodHandler[method],
		}

		c <- j

	}

	wait.Done()

}






