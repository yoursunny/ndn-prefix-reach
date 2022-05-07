package main

import (
	"net"
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

var connections sync.Map

func connect(ni nodeInfo) (fw l3.Forwarder, e error) {
	if stored, ok := connections.Load(ni.Host); ok {
		return stored.(l3.Forwarder), nil
	}

	tr, e := sockettransport.Dial("udp", ":0", net.JoinHostPort(ni.Host, "6363"), sockettransport.Config{
		MTU: 1280,
	})
	if e != nil {
		return nil, e
	}

	l3face, e := l3.NewFace(tr, l3.FaceConfig{})
	if e != nil {
		tr.Close()
		return nil, e
	}

	fw = l3.NewForwarder()
	face, e := fw.AddFace(l3face)
	if e != nil {
		close(l3face.Tx())
		return nil, e
	}
	face.AddRoute(ndn.Name{})

	stored, loaded := connections.LoadOrStore(ni.Host, fw)
	if loaded { // another goroutine made a Forwarder, use the existing one
		face.Close()
	}
	return stored.(l3.Forwarder), nil
}
