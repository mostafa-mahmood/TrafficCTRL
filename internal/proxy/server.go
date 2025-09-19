package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/mostafa-mahmood/TrafficCTRL/internal/logger"
	"go.uber.org/zap"
)

func StartServer(port uint16, targetUrl *url.URL, lgr logger.Logger) error {
	lgr.Info("TrafficCTRL started succesfully", zap.Uint16("proxy_port", port),
		zap.String("target_url", targetUrl.String()))

	proxy := createProxy(targetUrl)

	mux := http.NewServeMux()

	mux.Handle("/", proxy)

	address := net.JoinHostPort("", fmt.Sprintf("%d", port))
	return http.ListenAndServe(address, mux)
}
