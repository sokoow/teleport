/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/
package reversetunnel

import (
	"fmt"
	"net"
	"time"

	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/teleport/lib/utils/proxy"
	"github.com/gravitational/trace"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// peeredCluster is a remote cluster that has established
// a tunnel to the peers
type peeredCluster struct {
	log         *log.Entry
	clusterName string
	accessPoint auth.AccessPoint
	connInfo    services.TunnelConnection
}

func (s *peeredCluster) CachingAccessPoint() (auth.AccessPoint, error) {
	return s.accessPoint, nil
}

func (s *peeredCluster) GetClient() (auth.ClientI, error) {
	return s.clt, nil
}

func (s *peeredCluster) String() string {
	return fmt.Sprintf("peeredCluster(%v)", s.clusterName)
}

func (s *peeredCluster) GetStatus() string {
	diff := time.Now().Sub(s.connInfo.GetLastHeartbeat())
	if diff > defaults.ReverseTunnelOfflineThreshold {
		return RemoteSiteStatusOffline
	}
	return RemoteSiteStatusOnline
}

func (s *peeredCluster) GetName() string {
	return s.clusterName
}

func (s *peeredCluster) GetLastConnected() time.Time {
	return s.connInfo.GetLastHeartbeat()
}

// Dial is used to connect a requesting client (say, tsh) to an SSH server
// located in a remote connected site, the connection goes through the
// reverse proxy tunnel.
func (s *peeredCluster) Dial(from, to net.Addr) (conn net.Conn, err error) {
	s.log.Infof("[TUNNEL] forward peer %v@%v through the peer %v", to, s.clusterName, s.connInfo.GetProxyAddr())

	// "proxy:host:22@clustername"
	_, addr := to.Network(), to.String()

	client, err := proxy.DialWithDeadline("tcp", s.connInfo.GetProxyAddr(), &ssh.ClientConfig{})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer client.Close()

	se, err := client.NewSession()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer se.Close()

	writer, err := se.StdinPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	reader, err := se.StdoutPipe()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// Request opening TCP connection to the remote host
	if err := se.RequestSubsystem(fmt.Sprintf("proxy:%v@%v", to, s.clusterName)); err != nil {
		return nil, trace.Wrap(err)
	}

	return utils.NewPipeNetConn(
		reader,
		writer,
		se,
		from,
		to,
	), nil
}

// sortedPeeredClusters sorts peered clusters by the heartbeat time
type sortedPeeredClusters []*peeredCluster

// Len returns length of a role list
func (s sortedPeeredClusters) Len() int {
	return len(s)
}

// Less stacks latest attempts to the end of the list
func (s sortedPeeredClusters) Less(i, j int) bool {
	return s[i].connInfo.GetLastHeartbeat().After(s[j].connInfo.GetLastHeartbeat())
}

// Swap swaps two attempts
func (s sortedPeeredClusters) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
