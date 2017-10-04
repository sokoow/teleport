package reversetunnel

import (
	"net"

	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/teleport/lib/utils/proxy"

	"github.com/gravitational/trace"
	"golang.org/x/crypto/ssh"
)

// DialFn is a dialer that dials remote address, source
// address is used to preserve original address
type DialFn func(from net.Addr, to net.Addr) (net.Conn, error)

// NewProxyDialer creates a dialer that accesses auth server via remote proxy
func NewProxyDialer(cfg ProxyDialerConfig) (*ProxyDialer, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	return &ProxyDialer{ProxyDialerConfig: cfg}, nil
}

// ProxyDialerConfig specifies configuration parameters for ProxyDialer
type ProxyDialerConfig struct {
	// Config is SSH client config
	Config *ssh.ClientConfig
	// Dial is dialer function to connect to remote auth server
	// via proxy
	Dial DialFn
}

// CheckAndSetDefaults checks and sets default values
func (cfg *ProxyDialerConfig) CheckAndSetDefaults() error {
	if cfg.Dial == nil {
		return trace.BadParameter("missing parameter Dial")
	}
	if cfg.Config == nil {
		return trace.BadParameter("missing parameter Config")
	}
	if cfg.Config.User == "" {
		return trace.BadParameter("missing parameter Config.User")
	}
	if cfg.Config.Timeout == 0 {
		cfg.Config.Timeout = defaults.DefaultDialTimeout
	}
	return nil
}

// ProxyDialer uses remote SSH proxy to dial remote auth server
type ProxyDialer struct {
	ProxyDialerConfig
}

// Dial dials to remote auth server (ignoring addr that is passed as a parameter)
func (p *ProxyDialer) Dial(network, addr string) (net.Conn, error) {
	// we tell the remote site we want a connection to the auth server by using
	// a non-resolvable string @remote-auth-server as the destination.
	srcAddr := utils.NetAddr{Addr: "reversetunnel.proxy-dialer", AddrNetwork: "tcp"}
	dstAddr := utils.NetAddr{Addr: RemoteAuthServer, AddrNetwork: "tcp"}

	// first get a net.Conn (tcp connection) to the remote auth server. no
	// authentication will occur to this point
	netConn, err := p.ProxyDialerConfig.Dial(&srcAddr, &dstAddr)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// this will perform authentication against auth server via SSH
	client, err := proxy.NewClientConnWithDeadline(
		netConn, RemoteAuthServer, p.Config)
	if err != nil {
		netConn.Close()
		return nil, trace.Wrap(err)
	}

	// Dial again to HTTP server (auth server will interpret this request
	// and will give access to the socket)
	conn, err := client.Dial(network, addr)
	if err != nil {
		netConn.Close()
		client.Close()
		return nil, trace.Wrap(err)
	}

	// wrap all associated connections and add to the resulting connection
	// to release the resources on connection close
	return utils.NewCloserConn(conn, netConn, client), nil
}
