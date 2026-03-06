package trojan

import (
	"encoding/binary"
	"io"
	"net"

	"Hamburger/exp/trojan/common"
	"Hamburger/exp/trojan/core/tunnel"
	"Hamburger/exp/trojan/log"
)

type PacketConn struct {
	tunnel.Conn
}

func (c *PacketConn) ReadFrom(payload []byte) (int, net.Addr, error) {
	return c.ReadWithMetadata(payload)
}

func (c *PacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	address, err := tunnel.NewAddressFromAddr("udp", addr.String())
	if err != nil {
		return 0, err
	}
	m := &tunnel.Metadata{
		Address: address,
	}
	return c.WriteWithMetadata(payload, m)
}

func (c *PacketConn) WriteWithMetadata(payload []byte, metadata *tunnel.Metadata) (int, error) {
	length := len(payload)
	lengthBuf := [2]byte{}
	var packetBuf [512]byte
	packet := packetBuf[:0]
	var err error
	packet, err = appendAddressBytes(packet, metadata.Address)
	if err != nil {
		return 0, err
	}
	binary.BigEndian.PutUint16(lengthBuf[:], uint16(length))
	packet = append(packet, lengthBuf[:]...)
	packet = append(packet, '\r', '\n')
	if len(payload) > 0 {
		buffers := net.Buffers{packet, payload}
		_, err = buffers.WriteTo(c.Conn)
	} else {
		_, err = c.Conn.Write(packet)
	}

	log.Debug("udp packet remote", c.RemoteAddr(), "metadata", metadata, "size", length)
	return len(payload), err
}

func (c *PacketConn) ReadWithMetadata(payload []byte) (int, *tunnel.Metadata, error) {
	addr := &tunnel.Address{
		NetworkType: "udp",
	}
	if err := addr.ReadFrom(c.Conn); err != nil {
		return 0, nil, common.NewError("failed to parse udp packet addr").Base(err)
	}
	lengthBuf := [2]byte{}
	if _, err := io.ReadFull(c.Conn, lengthBuf[:]); err != nil {
		return 0, nil, common.NewError("failed to read length")
	}
	length := int(binary.BigEndian.Uint16(lengthBuf[:]))

	crlf := [2]byte{}
	if _, err := io.ReadFull(c.Conn, crlf[:]); err != nil {
		return 0, nil, common.NewError("failed to read crlf")
	}

	if len(payload) < length || length > MaxPacketSize {
		io.CopyN(io.Discard, c.Conn, int64(length))
		return 0, nil, common.NewError("incoming packet size is too large")
	}

	if _, err := io.ReadFull(c.Conn, payload[:length]); err != nil {
		return 0, nil, common.NewError("failed to read payload")
	}

	log.Debug("udp packet from", c.RemoteAddr(), "metadata", addr.String(), "size", length)
	return length, &tunnel.Metadata{
		Address: addr,
	}, nil
}

func appendAddressBytes(buf []byte, address *tunnel.Address) ([]byte, error) {
	buf = append(buf, byte(address.AddressType))
	switch address.AddressType {
	case tunnel.DomainName:
		buf = append(buf, byte(len(address.DomainName)))
		buf = append(buf, address.DomainName...)
	case tunnel.IPv4:
		ip := address.IP.To4()
		if ip == nil {
			return nil, common.NewError("invalid IPv4 address")
		}
		buf = append(buf, ip...)
	case tunnel.IPv6:
		ip := address.IP.To16()
		if ip == nil {
			return nil, common.NewError("invalid IPv6 address")
		}
		buf = append(buf, ip...)
	default:
		return nil, common.NewError("invalid address type")
	}
	var port [2]byte
	binary.BigEndian.PutUint16(port[:], uint16(address.Port))
	buf = append(buf, port[:]...)
	return buf, nil
}
