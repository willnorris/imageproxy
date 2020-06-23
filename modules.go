package imageproxy

import "net/http"

// Module needs docs.
type Module interface {
	ImageproxyModule() ModuleInfo
}

// ModuleInfo needs docs.
type ModuleInfo struct {
	ID ModuleID

	New func() Module
}

// ModuleID needs docs.
type ModuleID string

// A RequestAuthorizer determines if a request is authorized to be processed.
// Requests are processed before the remote resource is retrieved.
type RequestAuthorizer interface {
	// Authorize returns an error if the request should not
	// be processed further (for example, it doesn't have a
	// valid signature, is not for an allowed host, etc).
	AuthorizeRequest(req *http.Request) error
}

// A ResponseAuthorizer determines if a response from a remote server
// is authorized to be returned.
type ResponseAuthorizer interface {
	// AuthorizeResponse returns an error if a response should not be
	// returned to a client (for example, it is not for an image
	// resource, etc).
	AuthorizeResponse(res http.Response) error
}
