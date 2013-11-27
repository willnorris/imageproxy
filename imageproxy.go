package main

import (
	"log"
	"net/http"

	"github.com/willnorris/go-imageproxy/proxy"
)

func main() {
	p := proxy.NewProxy(nil)
	server := &http.Server{
		Addr:    ":8080",
		Handler: p,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
