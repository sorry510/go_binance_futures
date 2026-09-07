package llm

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHTTPTransportUsesSOCKS5HAndDelegatesDNS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	target := make(chan string, 1)
	go serveSingleSOCKS5Handshake(t, listener, target)

	transport, err := newHTTPTransport(Config{Timeout: time.Second, ProxyURL: "socks5h://" + listener.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	httpTransport, ok := transport.client.Transport.(*http.Transport)
	if !ok || httpTransport.DialContext == nil {
		t.Fatalf("unexpected transport: %T", transport.client.Transport)
	}
	conn, err := httpTransport.DialContext(context.Background(), "tcp", "example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	select {
	case got := <-target:
		if got != "example.com:443" {
			t.Fatalf("SOCKS target=%q want example.com:443", got)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not receive target")
	}
}

func serveSingleSOCKS5Handshake(t *testing.T, listener net.Listener, target chan<- string) {
	t.Helper()
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}
	_, _ = conn.Write([]byte{0x05, 0x00})
	request := make([]byte, 4)
	if _, err := io.ReadFull(reader, request); err != nil || request[3] != 0x03 {
		return
	}
	length, err := reader.ReadByte()
	if err != nil {
		return
	}
	host := make([]byte, int(length))
	if _, err := io.ReadFull(reader, host); err != nil {
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBytes)
	target <- net.JoinHostPort(string(host), fmt.Sprint(port))
	_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0})
}

func TestOpenAIClientGenerateActuallyUsesSOCKS5HProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"ok","model":"test-model","choices":[{"message":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	target := make(chan string, 1)
	go serveSingleSOCKS5Bridge(t, listener, backendURL.Host, target)

	cfg := Config{
		Provider: ProviderOpenAICompatible,
		Model:    "test-model",
		APIURL:   "http://llm-test.invalid:" + backendURL.Port() + "/v1/chat/completions",
		ProxyURL: "socks5h://" + listener.Addr().String(),
		Timeout:  2 * time.Second,
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Generate(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "test"}}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != "OK" {
		t.Fatalf("content=%q", response.Content)
	}
	diagnostics := ClientProxyDiagnostics(client)
	if !diagnostics.Enabled || !diagnostics.Dialed {
		t.Fatalf("expected actual SOCKS5H dial, got %+v", diagnostics)
	}
	select {
	case got := <-target:
		want := "llm-test.invalid:" + backendURL.Port()
		if got != want {
			t.Fatalf("SOCKS target=%q want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not receive LLM request target")
	}
}

func serveSingleSOCKS5Bridge(t *testing.T, listener net.Listener, backendAddress string, target chan<- string) {
	t.Helper()
	clientConn, err := listener.Accept()
	if err != nil {
		return
	}
	defer clientConn.Close()
	reader := bufio.NewReader(clientConn)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}
	_, _ = clientConn.Write([]byte{0x05, 0x00})
	request := make([]byte, 4)
	if _, err := io.ReadFull(reader, request); err != nil || request[3] != 0x03 {
		return
	}
	length, err := reader.ReadByte()
	if err != nil {
		return
	}
	host := make([]byte, int(length))
	if _, err := io.ReadFull(reader, host); err != nil {
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBytes)
	target <- net.JoinHostPort(string(host), fmt.Sprint(port))

	backendConn, err := net.DialTimeout("tcp", backendAddress, time.Second)
	if err != nil {
		return
	}
	defer backendConn.Close()
	_, _ = clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0})
	go func() { _, _ = io.Copy(backendConn, reader) }()
	_, _ = io.Copy(clientConn, backendConn)
}
