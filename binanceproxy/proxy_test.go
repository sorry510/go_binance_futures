package binanceproxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestNewParsesCommaSeparatedSOCKS5ProxyList(t *testing.T) {
	pool, err := New(" socks5://127.0.0.1:7890, socks5h://user:pass@proxy.example.com:1080 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(pool.proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(pool.proxies))
	}
	if len(pool.httpTransports) != 2 {
		t.Fatalf("expected 2 HTTP transports, got %d", len(pool.httpTransports))
	}
}

func TestNewAcceptsHTTPAndSOCKS5ProxySchemes(t *testing.T) {
	_, err := New("http://127.0.0.1:7890,https://127.0.0.1:7891,socks5://127.0.0.1:1080,socks5h://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsUnsupportedProxyScheme(t *testing.T) {
	if _, err := New("ftp://127.0.0.1:21"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}

func TestHTTPClientUsesRoundRobinTransport(t *testing.T) {
	pool, err := New("socks5://127.0.0.1:7890,socks5://127.0.0.1:7891")
	if err != nil {
		t.Fatal(err)
	}
	client := pool.HTTPClient()
	transport, ok := client.Transport.(*roundRobinTransport)
	if !ok {
		t.Fatalf("expected roundRobinTransport, got %T", client.Transport)
	}
	if len(transport.transports) != 2 {
		t.Fatalf("expected 2 transports, got %d", len(transport.transports))
	}
}

func TestWebsocketProxyURLNormalizesSOCKS5H(t *testing.T) {
	proxyURL, _ := url.Parse("socks5h://user:pass@127.0.0.1:1080")
	got := websocketProxyURL(proxyURL)
	want := "socks5://user:pass@127.0.0.1:1080"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestWithWSRotatesProxyOnReconnect(t *testing.T) {
	pool, err := New("socks5://127.0.0.1:7890,socks5://127.0.0.1:7891")
	if err != nil {
		t.Fatal(err)
	}
	var selected []string
	connect := func() (chan struct{}, chan struct{}, error) {
		return make(chan struct{}), make(chan struct{}), nil
	}
	for i := 0; i < 3; i++ {
		_, _, err := pool.withWS(func(proxyURL string) { selected = append(selected, proxyURL) }, connect)
		if err != nil {
			t.Fatal(err)
		}
	}
	if selected[0] != "socks5://127.0.0.1:7890" || selected[1] != "socks5://127.0.0.1:7891" || selected[2] != selected[0] {
		t.Fatalf("unexpected websocket proxy rotation: %v", selected)
	}
}

func TestSOCKS5HTTPTransportDoesNotUseHTTPProxyField(t *testing.T) {
	pool, err := New("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := pool.httpTransports[0].(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", pool.httpTransports[0])
	}
	if transport.Proxy != nil {
		t.Fatal("SOCKS5 transport must use DialContext instead of HTTP Proxy")
	}
	if transport.DialContext == nil {
		t.Fatal("SOCKS5 transport must configure DialContext")
	}
}
