package cloud

import "net"

type CloudInstance struct {
	Name       string
	PublicIPv4 net.IP
	PublicIPv6 net.IP
	Tags       map[string]string
}
