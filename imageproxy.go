package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/willnorris/go-imageproxy/cache"
	"github.com/willnorris/go-imageproxy/proxy"
)

var port = flag.Int("port", 8080, "port to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")

func main() {
	flag.Parse()

	fmt.Printf("go-imageproxy listening on port %d\n", *port)

	p := proxy.NewProxy(nil)
	p.Cache = cache.NewMemoryCache()
	if *whitelist != "" {
		p.Whitelist = strings.Split(*whitelist, ",")
	}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: p,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
