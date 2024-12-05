// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package stdnet implements the transport.Net interface
// using methods from Go's standard net package.
package stdnet

import (
	"fmt"
	"log"
	"net"
	"runtime/debug"
	"time"

	"github.com/pion/transport/v2"
	"github.com/wlynxg/anet"
	"github.com/ziti-livekit-example/lib/openziti"
)

const (
	lo0String = "lo0String"
	udpString = "udp"
)

// Net is an implementation of the net.Net interface
// based on functions of the standard net package.
type Net struct {
	interfaces []*transport.Interface
}

// NewNet creates a new StdNet instance.
func NewNet() (*Net, error) {
	n := &Net{}

	return n, n.UpdateInterfaces()
}

// Compile-time assertion
var _ transport.Net = &Net{}

// UpdateInterfaces updates the internal list of network interfaces
// and associated addresses.
func (n *Net) UpdateInterfaces() error {
	ifs := []*transport.Interface{}

	oifs, err := anet.Interfaces()
	if err != nil {
		return err
	}

	for i := range oifs {
		ifc := transport.NewInterface(oifs[i])

		addrs, err := anet.InterfaceAddrsByInterface(&oifs[i])
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			ifc.AddAddress(addr)
		}

		ifs = append(ifs, ifc)
	}

	n.interfaces = ifs

	return nil
}

// Interfaces returns a slice of interfaces which are available on the
// system
func (n *Net) Interfaces() ([]*transport.Interface, error) {
	return n.interfaces, nil
}

// InterfaceByIndex returns the interface specified by index.
//
// On Solaris, it returns one of the logical network interfaces
// sharing the logical data link; for more precision use
// InterfaceByName.
func (n *Net) InterfaceByIndex(index int) (*transport.Interface, error) {
	for _, ifc := range n.interfaces {
		if ifc.Index == index {
			return ifc, nil
		}
	}

	return nil, fmt.Errorf("%w: index=%d", transport.ErrInterfaceNotFound, index)
}

// InterfaceByName returns the interface specified by name.
func (n *Net) InterfaceByName(name string) (*transport.Interface, error) {
	for _, ifc := range n.interfaces {
		if ifc.Name == name {
			return ifc, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", transport.ErrInterfaceNotFound, name)
}

// func AnotherDialTest(address string) {
// 	fallback := &openziti.FallbackDialer{
// 		UnderlayDialer: &net.Dialer{},
// 	}
// 	dialer := openziti.ZitiContexts.NewDialerWithFallback(context.Background(), fallback)

// 	// Dial the Ziti service
// 	log.Print("tesssstsss ", address)
// 	network := "udp4"
// 	conn, err := dialer.Dial(network, address)
// 	if err != nil {
// 		log.Print("testeerrrrrrrr ", err)
// 	}

// 	zzcon := ZitiPacketConn{ZitiTURN: conn, network: network}

// 	for {
// 		b := make([]byte, 1)
// 		n, err := zzcon.ZitiTURN.Read(b)
// 		if err != nil {
// 			log.Print("testooooooo ", err)
// 		}

// 		log.Print("testpackeeeets ", string(n), " - ", string(b), " ")
// 	}
// }

var allConns []*ZitiPacketConns

type ZitiPacketConns struct {
	conns []*ZitiPacketConn
}

type ZitiPacketConn struct {
	zitiCon net.Conn
	network string
	address net.Addr
}

func (z *ZitiPacketConns) ReadFrom(b []byte) (int, net.Addr, error) {
	tries := 0
	for {
		for _, zc := range z.conns {
			connsStr := ""
			for _, zcs := range allConns {
				for _, zc := range zcs.conns {
					connsStr += " " + zc.address.String()
				}
			}
			// log.Printf("Amount of conns %d in (%d) - %s - %s, all available: %s",
			// 	len(z.conns), len(allConns), zc.address.String(), zc.network, connsStr)

			// Read data from the Ziti connection
			zc.zitiCon.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := zc.zitiCon.Read(b)
			if err != nil {
				if err.Error() != "read timed out" {
					log.Print("sssssssssssssssss ", err)
				}
				continue
			}
			log.Print("PACKEEEEEEEEEEEEET ", zc.address.String(), " ", string(n), " ")
			return n, zc.address, err
		}
		tries += 1
		if tries == 10 {
			log.Print("tries exceeded 10, running again")
			return z.ReadFrom(b)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// s, addr, err := z.net.ReadFrom(b)
	// log.Printf("OLDPACKEEEET %d %s ", s, addr)
	// return s, addr, err
}

func (z *ZitiPacketConns) WriteTo(b []byte, addr net.Addr) (int, error) {
	// Write data to the Ziti connection
	log.Print(string(debug.Stack()))
	log.Print("writeTo: ", addr.String())

	var fCon *ZitiPacketConn
	for _, zc := range z.conns {
		if zc.address.String() == addr.String() {
			fCon = zc
		}
	}

	if fCon == nil {
		err := fmt.Errorf("address not found %s", addr.String())
		log.Print(err)
		return 0, err
	}

	log.Print("WRIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIITE TO ", addr, ", length: ", len(b))

	i, err := fCon.zitiCon.Write(b)
	if err != nil {
		log.Print("lllllllllllllllllllllll", err)
	}
	log.Printf("INFOOOOOS %d", i)
	// i, err := z.net.WriteTo(b, addr)
	return i, err
}

func (z *ZitiPacketConns) Close() error {
	// Close the Ziti session
	log.Print("closing all connections")
	for _, zc := range z.conns {
		err := zc.zitiCon.Close()
		if err != nil {
			return fmt.Errorf("failed to close ziticonn for address %s, error: %s", zc.address.String(), err)
		}
	}

	return nil
}

func (z *ZitiPacketConns) LocalAddr() net.Addr {
	// Return a placeholder address; Ziti abstracts this
	return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
}

func (z *ZitiPacketConns) SetDeadline(t time.Time) error {
	for _, zc := range z.conns {
		err := zc.zitiCon.SetDeadline(t)
		if err != nil {
			return fmt.Errorf("failed to setdeadline for address %s, error: %s", zc.address.String(), err)
		}
	}

	return nil
}

func (z *ZitiPacketConns) SetReadDeadline(t time.Time) error {
	for _, zc := range z.conns {
		err := zc.zitiCon.SetReadDeadline(t)
		if err != nil {
			return fmt.Errorf("failed to SetReadDeadline for address %s, error: %s", zc.address.String(), err)
		}
	}

	return nil
}

func (z *ZitiPacketConns) SetWriteDeadline(t time.Time) error {
	for _, zc := range z.conns {
		err := zc.zitiCon.SetWriteDeadline(t)
		if err != nil {
			return fmt.Errorf("failed to SetWriteDeadline for address %s, error: %s", zc.address.String(), err)
		}
	}

	return nil
}

func (z *ZitiPacketConns) AppendConn(address net.Addr, network string) error {
	// fallback := &openziti.FallbackDialer{
	// 	UnderlayDialer: &net.Dialer{},
	// }
	// dialer := openziti.ZitiContexts.NewDialerWithFallback(context.Background(), fallback)
	dialer := openziti.ZitiContexts.NewDialer()

	log.Print("AppendConn ", address)
	conn, err := dialer.Dial(network, address.String())
	if err != nil {
		log.Print("AppendConn error ", err)
		return err
	}
	z.conns = append(z.conns, &ZitiPacketConn{zitiCon: conn, network: network, address: address})

	return nil
}

// ListenPacket announces on the local network address.
func (n *Net) ListenPacket(network string, address string) (net.PacketConn, error) {
	// fallback := &openziti.FallbackDialer{
	// 	UnderlayDialer: &net.Dialer{},
	// }
	// dialer := openziti.ZitiContexts.NewDialerWithFallback(context.Background(), fallback)

	dialer := openziti.ZitiContexts.NewDialer()

	// Dial the Ziti service
	log.Print("beforeiiii ", address, network)
	if len(address) < 11 {
		address = "12.34.56.78:3478"
	}

	udpAddr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		log.Println("Error resolving UDP address:", err)
		return nil, err
	}

	log.Print("lililililili ", address, " - ", network)
	conn, err := dialer.Dial(network, address)
	if err != nil {
		log.Print("Anomalllyyy ", err)
		return nil, err
	}
	log.Print("4444444444444444")

	// ne, err := net.ListenPacket(network, address)
	// if err != nil {
	// 	log.Print(err)
	// }

	zpc := &ZitiPacketConn{zitiCon: conn, network: network, address: udpAddr}
	zpcs := &ZitiPacketConns{conns: []*ZitiPacketConn{zpc}}
	allConns = append(allConns, zpcs)
	return zpcs, nil

	// return net.ListenPacket(network, address)
}

type zitiUdp struct {
	transport.UDPConn
	net transport.UDPConn
}

func (u zitiUdp) Write(b []byte) (int, error) {
	log.Print("qqqqqqqqqqqqqqqqq")
	return u.net.Write(b)
}

func (u zitiUdp) WriteTo(b []byte, n net.Addr) (int, error) {
	log.Print("qqqqqqqqqqqqq22222222222222")
	return u.net.WriteTo(b, n)
}

func (u zitiUdp) WriteMsgUDP(b []byte, b2 []byte, n *net.UDPAddr) (int, int, error) {
	log.Print("qqqqqqqqqqqqqqqqq33333333333333")
	return u.net.WriteMsgUDP(b, b2, n)
}

func (u zitiUdp) WriteToUDP(b []byte, n *net.UDPAddr) (int, error) {
	log.Print("qqqqqqqqqqqqqqqqq44444444444")
	return u.net.WriteToUDP(b, n)
}

func (u zitiUdp) Read(b []byte) (int, error) {
	log.Print("qqqqqqqqqqqqqqqqq555555555555")
	return u.net.Read(b)
}

func (u zitiUdp) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	log.Print("qqqqqqqqqqqqqqqqq666666666666")
	return u.net.ReadFrom(p)
}

func (u zitiUdp) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	log.Print("qqqqqqqqqqqqqqqqq777777777777777")
	return u.net.ReadFromUDP(b)
}

func (u zitiUdp) ReadMsgUDP(b []byte, oob []byte) (n int, oobn int, flags int, addr *net.UDPAddr, err error) {
	log.Print("qqqqqqqqqqqqqqqqq8888888888888888")
	return u.net.ReadMsgUDP(b, oob)
}

// ListenUDP acts like ListenPacket for UDP networks.
func (n *Net) ListenUDP(network string, locAddr *net.UDPAddr) (transport.UDPConn, error) {
	log.Print("33333333333333 ", locAddr, " ")
	var err error
	z := zitiUdp{}
	z.net, err = net.ListenUDP(network, locAddr)
	return z, err
}

// Dial connects to the address on the named network.
func (n *Net) Dial(network, address string) (net.Conn, error) {
	log.Print("2222222222222 ", address)
	return net.Dial(network, address)
}

// DialUDP acts like Dial for UDP networks.
func (n *Net) DialUDP(network string, laddr, raddr *net.UDPAddr) (transport.UDPConn, error) {
	log.Print("11111111111111")
	return net.DialUDP(network, laddr, raddr)
}

// ResolveIPAddr returns an address of IP end point.
func (n *Net) ResolveIPAddr(network, address string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(network, address)
}

// ResolveUDPAddr returns an address of UDP end point.
func (n *Net) ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr(network, address)
}

// ResolveTCPAddr returns an address of TCP end point.
func (n *Net) ResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	log.Print(network)
	return net.ResolveTCPAddr(network, address)
}

// DialTCP acts like Dial for TCP networks.
func (n *Net) DialTCP(network string, laddr, raddr *net.TCPAddr) (transport.TCPConn, error) {
	log.Print("555555555555555555555555 ", laddr.String(), " - ", raddr.String())
	return net.DialTCP(network, laddr, raddr)
}

// ListenTCP acts like Listen for TCP networks.
func (n *Net) ListenTCP(network string, laddr *net.TCPAddr) (transport.TCPListener, error) {
	log.Print("777777777777777")
	l, err := net.ListenTCP(network, laddr)
	if err != nil {
		return nil, err
	}

	return tcpListener{l}, nil
}

type tcpListener struct {
	*net.TCPListener
}

func (l tcpListener) AcceptTCP() (transport.TCPConn, error) {
	return l.TCPListener.AcceptTCP()
}

type stdDialer struct {
	*net.Dialer
}

func (d stdDialer) Dial(network, address string) (net.Conn, error) {
	log.Print("6666666666666")
	return d.Dialer.Dial(network, address)
}

// CreateDialer creates an instance of vnet.Dialer
func (n *Net) CreateDialer(d *net.Dialer) transport.Dialer {
	return stdDialer{d}
}
