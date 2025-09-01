package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
)

func StartServer(port uint16, targetUrl *url.URL) error {
	proxy := CreateProxy(targetUrl)

	mux := http.NewServeMux()

	mux.Handle("/", proxy)

	address := net.JoinHostPort("", fmt.Sprintf("%d", port))
	return http.ListenAndServe(address, mux)
}
