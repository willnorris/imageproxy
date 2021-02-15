module willnorris.com/go/imageproxy

require (
	cloud.google.com/go v0.76.0 // indirect
	cloud.google.com/go/storage v1.13.0
	github.com/Azure/azure-sdk-for-go v51.1.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/PaulARoy/azurestoragecache v0.0.0-20170906084534-3c249a3ba788
	github.com/aws/aws-sdk-go v1.37.10
	github.com/die-net/lrucache v0.0.0-20190707192454-883874fe3947
	github.com/disintegration/imaging v1.6.2
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/fcjr/aia-transport-go v1.2.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/jamiealquiza/envy v1.1.0
	github.com/muesli/smartcrop v0.3.0
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/peterbourgon/diskv v0.0.0-20171120014656-2973218375c3
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd
	github.com/satori/go.uuid v0.0.0-20180103174451-36e9d2ebbde5 // indirect
	go.opencensus.io v0.22.6 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/image v0.0.0-20201208152932-35266b937fa6
	golang.org/x/oauth2 v0.0.0-20210210192628-66670185b0cd // indirect
	google.golang.org/api v0.39.0 // indirect
	google.golang.org/genproto v0.0.0-20210211154401-3a9a48ddfd6c // indirect
	willnorris.com/go/gifresize v1.0.0
)

// local copy of envy package without cobra support
replace github.com/jamiealquiza/envy => ./third_party/envy

go 1.13
