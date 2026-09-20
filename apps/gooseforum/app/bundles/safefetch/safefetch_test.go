package safefetch

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeResolver map[string][]netip.Addr

func (r fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return addresses, nil
}

func publicAddr() netip.Addr { return netip.MustParseAddr("93.184.216.34") }

func mappedDialer(target string, calls *atomic.Int32) DialContextFunc {
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		calls.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, target)
	}
}

func TestFetchReturnsBoundedHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<title>Example</title>")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:    fakeResolver{"preview.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &calls),
	})
	result, err := client.Fetch(t.Context(), "http://preview.example/page#fragment")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.StatusCode != http.StatusOK || string(result.Body) != "<title>Example</title>" {
		t.Fatalf("Fetch() result = %#v", result)
	}
	if result.FinalURL.Fragment != "" {
		t.Fatalf("Fetch() final fragment = %q, want empty", result.FinalURL.Fragment)
	}
	if calls.Load() != 1 {
		t.Fatalf("dial calls = %d, want 1", calls.Load())
	}
}

func TestFetchRejectsPrivateDNSBeforeDial(t *testing.T) {
	var calls atomic.Int32
	client := New(Config{
		Resolver: fakeResolver{
			"private.example": {netip.MustParseAddr("127.0.0.1")},
			"localhost":       {netip.MustParseAddr("127.0.0.1")},
		},
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			calls.Add(1)
			return nil, errors.New("must not dial")
		},
	})
	for _, target := range []string{"http://private.example/", "http://localhost/"} {
		_, err := client.Fetch(t.Context(), target)
		if !IsClass(err, ErrorBlocked) {
			t.Fatalf("Fetch(%q) error = %v, want blocked", target, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("dial calls = %d, want 0", calls.Load())
	}
}

func TestFetchRejectsNonPublicLiteralTargetsBeforeDial(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/",
		"http://[::1]/",
		"http://10.0.0.1/",
		"http://172.16.0.1/",
		"http://192.168.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[fc00::1]/",
		"http://[fe80::1]/",
		"http://224.0.0.1/",
		"http://0.1.2.3/",
		"http://192.88.99.1/",
		"http://[64:ff9b:1::1]/",
		"http://[100::1]/",
		"http://[2001::1]/",
		"http://[2002::1]/",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			var calls atomic.Int32
			client := New(Config{DialContext: func(context.Context, string, string) (net.Conn, error) {
				calls.Add(1)
				return nil, errors.New("must not dial")
			}})
			_, err := client.Fetch(t.Context(), target)
			if !IsClass(err, ErrorBlocked) {
				t.Fatalf("Fetch() error = %v, want blocked", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("dial calls = %d, want 0", calls.Load())
			}
		})
	}
}

func TestFetchDialsOnlyTheValidatedAddress(t *testing.T) {
	var dialed string
	client := New(Config{
		Resolver: fakeResolver{"rebind.example": {publicAddr()}},
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, errors.New("stop after observing dial target")
		},
	})
	_, _ = client.Fetch(t.Context(), "https://rebind.example/article")
	if dialed != "93.184.216.34:443" {
		t.Fatalf("dial target = %q, want checked IP instead of hostname", dialed)
	}
}

func TestFetchRechecksRedirectDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://private.example/admin", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver: fakeResolver{
			"public.example":  {publicAddr()},
			"private.example": {netip.MustParseAddr("10.0.0.1")},
		},
		DialContext: mappedDialer(server.Listener.Addr().String(), &calls),
	})
	_, err := client.Fetch(t.Context(), "http://public.example/")
	if !IsClass(err, ErrorBlocked) {
		t.Fatalf("Fetch() redirect error = %v, want blocked", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("dial calls = %d, want only public hop", calls.Load())
	}
}

func TestFetchEnforcesRedirectLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := strings.Count(r.URL.Path, "next")
		if step < 4 {
			http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/")+"/next", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"redirect.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		MaxRedirects: 3,
	})
	_, err := client.Fetch(t.Context(), "http://redirect.example/")
	if !IsClass(err, ErrorRedirect) {
		t.Fatalf("Fetch() error = %v, want redirect", err)
	}
}

func TestFetchCapsDecompressedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(strings.Repeat("x", 1025)))
		_ = writer.Close()
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"large.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		MaxBodyBytes: 1024,
	})
	_, err := client.Fetch(t.Context(), "http://large.example/")
	if !IsClass(err, ErrorTooLarge) {
		t.Fatalf("Fetch() error = %v, want too_large", err)
	}
}

func TestFetchRejectsOversizedContentLengthBeforeReading(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "2048")
		_, _ = io.WriteString(w, strings.Repeat("x", 2048))
	}))
	t.Cleanup(server.Close)

	client := New(Config{
		Resolver:     fakeResolver{"large.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &atomic.Int32{}),
		MaxBodyBytes: 1024,
	})
	_, err := client.Fetch(t.Context(), "http://large.example/")
	if !IsClass(err, ErrorTooLarge) {
		t.Fatalf("Fetch() error = %v, want too_large", err)
	}
}

func TestFetchHonorsTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"slow.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		TotalTimeout: 20 * time.Millisecond,
	})
	_, err := client.Fetch(t.Context(), "http://slow.example/")
	if !IsClass(err, ErrorTimeout) {
		t.Fatalf("Fetch() error = %v, want timeout", err)
	}
}
