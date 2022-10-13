package server

import (
	"context"
	"github.com/Nixson/environment"
	"github.com/Nixson/http/session"
	"github.com/Nixson/logger"
	"net/http"
	"os"
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
	env := environment.GetEnv()
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handle))
	srv = &http.Server{
		Addr:           env.GetString("server.port"),
		Handler:        mux,
		MaxHeaderBytes: env.GetInt("server.maxSize"),
	}

}

func handle(w http.ResponseWriter, r *http.Request) {
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
	ctx.ParseUrl()
	if ctx.IsGranted() {
		ctx.Call()
	}
}
