package server

import (
	"context"
	"github.com/Nixson/environment"
	"github.com/Nixson/http/session"
	"github.com/Nixson/logger"
	"net/http"
	"os"
	"strings"
)

func RunWithSignal() {
	if srv == nil {
		InitServer()
	}
	done := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.Println("listen " + srv.Addr)
	go func() {
		_ = srv.ListenAndServe()
	}()
	<-done
	logger.Println("server done")
	_ = srv.Close()
	_ = srv.Shutdown(ctx)

}

func Run() {
	if srv == nil {
		InitServer()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.Println("listen " + srv.Addr)
	_ = srv.ListenAndServe()
	logger.Println("server done")
	_ = srv.Close()
	_ = srv.Shutdown(ctx)

}

var srv *http.Server

func InitServer() {
	logger.Println("init Server")
	env := environment.GetEnv()
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handle))
	mux.Handle("/static", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var path = request.URL.Path
		var sub = strings.Split(path, "/")
		logger.Println(sub)
		if sub[1] != "css" || sub[1] != "js" {
			path = strings.Join(sub[2:], "/")
		}
		logger.Println(path)
		http.ServeFile(writer, request, env.GetString("template.url")+path)
	}))
	srv = &http.Server{
		Addr:           env.GetString("server.port"),
		Handler:        mux,
		MaxHeaderBytes: env.GetInt("server.maxSize"),
	}

}

func handle(w http.ResponseWriter, r *http.Request) {
	env := environment.GetEnv()
	ctx := Context{
		Request:  r,
		Response: w,
		Session:  &session.Session{},
		Path:     r.URL.Path,
		Params:   make(map[string]string),
		Method:   r.Method,
		Data:     r.Body,
		Query:    r.URL.Query(),
	}
	static, ok := ctx.CheckStatic(env, r.URL.Path)
	if ok {
		http.ServeFile(w, r, env.GetString("template.url")+static)
		return
	}
	ctx.ParseUrl()
	if ctx.IsGranted() {
		ctx.Call()
	}
}
