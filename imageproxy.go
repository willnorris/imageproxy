package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/willnorris/go-imageproxy/cache"
	"github.com/willnorris/go-imageproxy/proxy"
)

var port = flag.Int("port", 8080, "port to listen on")

func main() {
	flag.Parse()

	fmt.Printf("go-imageproxy listening on port %d\n", *port)

	p := proxy.NewProxy(nil)
	p.Cache = cache.NewMemoryCache()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: p,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
