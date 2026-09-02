package binanceproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	spotbinance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
	xproxy "golang.org/x/net/proxy"
)

type Pool struct {
	proxies        []*url.URL
	httpTransports []http.RoundTripper
	next           atomic.Uint64
	wsMu           sync.Mutex
}

type roundRobinTransport struct {
	transports []http.RoundTripper
	next       atomic.Uint64
}

func New(raw string) (*Pool, error) {
	pool := &Pool{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		proxyURL, err := url.Parse(item)
		if err != nil || proxyURL.Host == "" {
			return nil, fmt.Errorf("invalid binance proxy %q", item)
		}
		switch strings.ToLower(proxyURL.Scheme) {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, fmt.Errorf("unsupported binance proxy scheme %q", proxyURL.Scheme)
		}

		transport, err := newHTTPTransport(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("create binance proxy %q: %w", item, err)
		}
		pool.proxies = append(pool.proxies, proxyURL)
		pool.httpTransports = append(pool.httpTransports, transport)
	}
	return pool, nil
}

func newHTTPTransport(proxyURL *url.URL) (http.RoundTripper, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5", "socks5h":
		transport.Proxy = nil
		forward := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		dialer, err := xproxy.FromURL(proxyURL, forward)
		if err != nil {
			return nil, err
		}
		contextDialer, ok := dialer.(xproxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("SOCKS5 dialer does not support context")
		}
		transport.DialContext = contextDialer.DialContext
	}
	return transport, nil
}

func (p *Pool) Enabled() bool {
	return p != nil && len(p.proxies) > 0
}

func (p *Pool) nextProxy() *url.URL {
	if !p.Enabled() {
		return nil
	}
	index := p.next.Add(1) - 1
	return p.proxies[index%uint64(len(p.proxies))]
}

func (p *Pool) HTTPClient() *http.Client {
	if !p.Enabled() {
		return http.DefaultClient
	}
	return &http.Client{Transport: &roundRobinTransport{transports: p.httpTransports}}
}

func (t *roundRobinTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	index := t.next.Add(1) - 1
	return t.transports[index%uint64(len(t.transports))].RoundTrip(req)
}

func (p *Pool) WithFuturesWS(connect func() (chan struct{}, chan struct{}, error)) (chan struct{}, chan struct{}, error) {
	return p.withWS(func(proxyURL string) { futures.SetWsProxyUrl(proxyURL) }, connect)
}

func (p *Pool) WithDeliveryWS(connect func() (chan struct{}, chan struct{}, error)) (chan struct{}, chan struct{}, error) {
	return p.withWS(func(proxyURL string) { delivery.SetWsProxyUrl(proxyURL) }, connect)
}

func (p *Pool) WithSpotWS(connect func() (chan struct{}, chan struct{}, error)) (chan struct{}, chan struct{}, error) {
	return p.withWS(func(proxyURL string) { spotbinance.SetWsProxyUrl(proxyURL) }, connect)
}

func (p *Pool) withWS(setProxy func(string), connect func() (chan struct{}, chan struct{}, error)) (chan struct{}, chan struct{}, error) {
	if !p.Enabled() {
		return connect()
	}
	p.wsMu.Lock()
	defer p.wsMu.Unlock()
	setProxy(websocketProxyURL(p.nextProxy()))
	return connect()
}

func websocketProxyURL(proxyURL *url.URL) string {
	clone := *proxyURL
	if strings.EqualFold(clone.Scheme, "socks5h") {
		clone.Scheme = "socks5"
	}
	return clone.String()
}
