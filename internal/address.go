package internal

import (
	"errors"
	"net"
	"strconv"
)

// Address is an IP:Port combination.
type Address struct {
	Host net.IP
	Port uint16
}

func (a Address) String() string {
	return net.JoinHostPort(a.Host.String(), strconv.Itoa(int(a.Port)))
}

func (a Address) IsValid() bool {
	return a.Port != 0 && (len(a.Host) == 16 || len(a.Host) == 4)
}

func ParseAddress(hostport string) (Address, error) {
	hosts, ports, err := net.SplitHostPort(hostport)
	if err != nil {
		return Address{}, err
	}
	host := net.ParseIP(hosts)
	if host == nil {
		return Address{}, errors.New("bad ip")
	}
	port, err := strconv.Atoi(ports)
	if err != nil {
		return Address{}, err
	}
	if port < 0 || port > 65535 {
		return Address{}, errors.New("range")
	}
	return Address{Host: host, Port: uint16(port)}, nil
}
