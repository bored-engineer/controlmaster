package controlmaster

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Dial runs net.Dial then NewClient on the result
func Dial(network string, address string) (*ssh.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	client, err := NewClient(conn)
	if err != nil {
		return nil, err
	}
	return client, nil	
}

// DialTimeout runs net.DialTimeout then NewClient on the result
func DialTimeout(network string, address string, duration time.Duration) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, address, duration)
	if err != nil {
		return nil, err
	}
	client, err := NewClient(conn)
	if err != nil {
		return nil, err
	}
	return client, nil	
}

// NewClient creates a *ssh.Client given a net.Conn
func NewClient(conn net.Conn) (*ssh.Client, error) {
	transport, err := handshakeControlProxy(conn)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConnFromTransport(transport)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}