package controlmaster

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	mux_MSG_HELLO = 0x00000001
	mux_C_PROXY   = 0x1000000f
	mux_S_PROXY   = 0x8000000f
	mux_S_FAILURE = 0x80000003
)

// handshakeControlProxy attempts to establish a transport connection with an
// ssh ControlMaster socket in proxy mode. For details see:
// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.mux
func handshakeControlProxy(rw io.ReadWriteCloser) (*transport, error) {
	b := &controlBuffer{}
	if err := b.WriteUint32(mux_MSG_HELLO); err != nil {
		return nil, err
	}
	// Protocol Version
	if err := b.WriteUint32(4); err != nil {
		return nil, err
	}
	tmp, err := b.LengthPrefixedBytes()
	if err != nil {
		return nil, err
	}
	if _, err := rw.Write(tmp); err != nil {
		return nil, fmt.Errorf("mux hello write failed: %v", err)
	}

	b.Reset()
	if err := b.WriteUint32(mux_C_PROXY); err != nil {
		return nil, err
	}
	// Request ID
	if err := b.WriteUint32(0); err != nil {
		return nil, err
	}
	tmp, err = b.LengthPrefixedBytes()
	if err != nil {
		return nil, err
	}
	if _, err := rw.Write(tmp); err != nil {
		return nil, fmt.Errorf("mux client proxy write failed: %v", err)
	}

	r := controlReader{rw}
	m, err := r.Next()
	if err != nil {
		return nil, fmt.Errorf("mux hello read failed: %v", err)
	}
	if m.messageType != mux_MSG_HELLO {
		return nil, fmt.Errorf("mux reply not hello")
	}
	if v, err := m.ReadUint32(); err != nil || v != 4 {
		return nil, fmt.Errorf("mux reply hello has bad protocol version")
	}
	m, err = r.Next()
	if err != nil {
		return nil, fmt.Errorf("error reading mux server proxy: %v", err)
	}
	if m.messageType != mux_S_PROXY {
		return nil, fmt.Errorf("expected server proxy response got %d", m.messageType)
	}
	return &transport{rw}, nil
}

// transport implements the ssh.Transport interface for
// ControlMaster connections. Each controlMessage has zero length padding and
// no MAC.
type transport struct {
	rw io.ReadWriteCloser
}

func (p *transport) Close() error {
	return p.rw.Close()
}

func (p *transport) getSessionID() []byte {
	return nil
}

func (p *transport) ReadPacket() ([]byte, error) {
	var l uint32
	err := binary.Read(p.rw, binary.BigEndian, &l)
	if err == nil {
		buf := &bytes.Buffer{}
		_, err = io.CopyN(buf, p.rw, int64(l))
		if err == nil {
			// Discard the padding byte.
			buf.ReadByte()
			return buf.Bytes(), nil
		}
	}
	return nil, err
}

func (p *transport) WritePacket(controlMessage []byte) error {
	l := uint32(len(controlMessage)) + 1
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, &l) // controlMessage Length.
	b.WriteByte(0)                        // Padding Length.
	b.Write(controlMessage)
	_, err := p.rw.Write(b.Bytes())
	return err
}

type controlBuffer struct {
	bytes.Buffer
}

func (b *controlBuffer) WriteUint32(i uint32) error {
	return binary.Write(b, binary.BigEndian, i)
}

func (b *controlBuffer) LengthPrefixedBytes() ([]byte, error) {
	b2 := &bytes.Buffer{}
	if err := binary.Write(b2, binary.BigEndian, uint32(b.Len())); err != nil {
		return nil, err
	}
	if _, err := b2.Write(b.Bytes()); err != nil {
		return nil, err
	}
	return b2.Bytes(), nil
}

type controlMessage struct {
	body        bytes.Buffer
	messageType uint32
}

func (p controlMessage) ReadUint32() (uint32, error) {
	var u uint32
	err := binary.Read(&p.body, binary.BigEndian, &u)
	return u, err
}

func (p controlMessage) ReadString() (string, error) {
	var l uint32
	err := binary.Read(&p.body, binary.BigEndian, &l)
	if err != nil {
		return "", fmt.Errorf("error reading string length: %v", err)
	}
	b := p.body.Next(int(l))
	if len(b) != int(l) {
		return string(b), fmt.Errorf("EOF on string read")
	}
	return string(b), nil
}

type controlReader struct {
	r io.Reader
}

func (r controlReader) Next() (*controlMessage, error) {
	p := &controlMessage{}
	var len uint32
	err := binary.Read(r.r, binary.BigEndian, &len)
	if err != nil {
		return nil, fmt.Errorf("error reading message length: %v", err)
	}
	_, err = io.CopyN(&p.body, r.r, int64(len))
	if err != nil {
		return nil, fmt.Errorf("error reading message payload: %v", err)
	}
	err = binary.Read(&p.body, binary.BigEndian, &p.messageType)
	if err != nil {
		return nil, fmt.Errorf("error reading message type: %v", err)
	}
	if p.messageType == mux_S_FAILURE {
		reason, _ := p.ReadString()
		return nil, fmt.Errorf("server failure: '%s'", reason)
	}
	return p, nil
}