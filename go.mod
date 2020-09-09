module willnorris.com/go/imageproxy

require (
	cloud.google.com/go v0.65.0
	cloud.google.com/go/storage v1.11.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0 // indirect
	git.apache.org/thrift.git v0.12.0 // indirect
	github.com/Azure/azure-sdk-for-go v46.1.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.5 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.3 // indirect
	github.com/PaulARoy/azurestoragecache v0.0.0-20170906084534-3c249a3ba788
	github.com/aws/aws-sdk-go v1.34.20
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/die-net/lrucache v0.0.0-20190707192454-883874fe3947
	github.com/disintegration/imaging v1.6.2
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/garyburd/redigo v1.6.2
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/grpc-ecosystem/grpc-gateway v1.14.8 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/jamiealquiza/envy v1.1.0
	github.com/marstr/guid v0.0.0-20170427235115-8bdf7d1a087c // indirect
	github.com/muesli/smartcrop v0.3.0
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/peterbourgon/diskv v0.0.0-20171120014656-2973218375c3
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.13.0 // indirect
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd
	github.com/satori/go.uuid v0.0.0-20180103174451-36e9d2ebbde5 // indirect
	golang.org/x/build v0.0.0-20190314133821-5284462c4bec // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/image v0.0.0-20200801110659-972c09e46d76
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200909081042-eff7692f9009 // indirect
	golang.org/x/tools v0.0.0-20200909210914-44a2922940c2 // indirect
	google.golang.org/api v0.31.0 // indirect
	google.golang.org/genproto v0.0.0-20200904004341-0bd0a958aa1d // indirect
	google.golang.org/grpc v1.32.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	willnorris.com/go/gifresize v1.0.0
)

replace (
	// replace git.apache.org with github.com/apache (which is the upstream master
	// anyway), since git.apache.org is offline. v0.12.0 is the latest release, but
	// go complains about "github.com/apache/thrift@v0.12.0 used for two different
	// module paths".  Instead we move one commit ahead.
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.1-0.20190107215100-e824efcb7935

	// temporary fix to https://github.com/golang/lint/issues/436 which still seems to be a problem
	github.com/golang/lint => github.com/golang/lint v0.0.0-20181217174547-8f45f776aaf1

	// local copy of envy package without cobra support
	github.com/jamiealquiza/envy => ./third_party/envy
)

go 1.13
