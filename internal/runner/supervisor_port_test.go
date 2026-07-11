package runner

import (
	"net"
	"testing"
)

func TestCheckPortAvailableUsesLoaderProtocol(t *testing.T) {
	udpListener, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	udpPort := udpListener.LocalAddr().(*net.UDPAddr).Port
	defer udpListener.Close()
	if err := checkPortAvailable("bedrock", udpPort); err == nil {
		t.Fatal("expected occupied UDP port to be rejected for Bedrock")
	}

	tcpListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	tcpPort := tcpListener.Addr().(*net.TCPAddr).Port
	defer tcpListener.Close()
	if err := checkPortAvailable("vanilla", tcpPort); err == nil {
		t.Fatal("expected occupied TCP port to be rejected for Java")
	}
}
