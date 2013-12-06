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

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")

func main() {
	flag.Parse()

	fmt.Printf("go-imageproxy listening on %s\n", *addr)

	p := proxy.NewProxy(nil)
	p.Cache = cache.NewMemoryCache()
	p.MaxWidth = 2000
	p.MaxHeight = 2000
	if *whitelist != "" {
		p.Whitelist = strings.Split(*whitelist, ",")
	}
	server := &http.Server{
		Addr:    *addr,
		Handler: p,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
