package console

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerSettings(t *testing.T) {
	address := "127.0.0.1:5234"
	handler := http.NewServeMux()
	srv := newHTTPServer(address, handler)

	if srv.Addr != address {
		t.Fatalf("Addr = %q, want %q", srv.Addr, address)
	}
	if srv.Handler != handler {
		t.Fatalf("Handler = %T, want the supplied handler", srv.Handler)
	}
	if srv.ReadTimeout != 10*time.Second {
		t.Fatalf("ReadTimeout = %s, want 10s", srv.ReadTimeout)
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want 5s", srv.ReadHeaderTimeout)
	}
	if srv.WriteTimeout != 10*time.Second {
		t.Fatalf("WriteTimeout = %s, want 10s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want 60s", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, 1<<20)
	}
}
