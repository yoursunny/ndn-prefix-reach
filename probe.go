package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	_ "github.com/usnistgov/ndn-dpdk/ndn/keychain" // recognize ValidityPeriod
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

type probeRequest struct {
	Name        ndn.Name
	AddSuffix   bool
	CanBePrefix bool
	MustBeFresh bool
}

func probeRequestFromQuery(query url.Values) (req probeRequest) {
	req.Name = ndn.ParseName(query.Get("name"))
	req.AddSuffix = query.Get("suffix") != ""
	req.CanBePrefix = query.Get("cbp") != ""
	req.MustBeFresh = query.Get("mbf") != ""
	return
}

func (req probeRequest) MakeInterest() (interest ndn.Interest) {
	if req.AddSuffix {
		var suffix [8]byte
		rand.Read(suffix[:])
		interest.Name = append(ndn.Name{}, req.Name...)
		interest.Name = append(interest.Name, ndn.MakeNameComponent(an.TtSequenceNumNameComponent, suffix[:]))
	} else {
		interest.Name = req.Name
	}
	interest.CanBePrefix = req.CanBePrefix
	interest.MustBeFresh = req.MustBeFresh
	return
}

type probeResult struct {
	OK    bool                    `json:"ok"`
	RTT   nnduration.Milliseconds `json:"rtt,omitempty"`
	Error string                  `json:"error,omitempty"`
}

func probe(ctx context.Context, ni nodeInfo, req probeRequest, res *probeResult) {
	tr, e := sockettransport.Dial("udp", ":0", ni.Host+":6363")
	if e != nil {
		res.Error = e.Error()
		return
	}

	face, e := l3.NewFace(tr)
	if e != nil {
		close(tr.Tx())
		res.Error = e.Error()
		return
	}
	defer close(face.Tx())

	interest := req.MakeInterest()
	t0 := time.Now()
	face.Tx() <- interest
	for {
		select {
		case <-ctx.Done():
			e := ctx.Err()
			if errors.Is(e, context.DeadlineExceeded) {
				res.Error = "timeout"
			} else {
				res.Error = ctx.Err().Error()
			}
			return
		case pkt := <-face.Rx():
			switch {
			case pkt.Data != nil && pkt.Data.CanSatisfy(interest):
				t1 := time.Now()
				res.OK = true
				res.RTT = nnduration.Milliseconds(t1.Sub(t0).Milliseconds())
				return
			case pkt.Nack != nil && pkt.Nack.Interest.Name.Equal(interest.Name):
				t1 := time.Now()
				res.RTT = nnduration.Milliseconds(t1.Sub(t0).Milliseconds())
				res.Error = "Nack " + an.NackReasonString(pkt.Nack.Reason)
				return
			}
		}
	}
}

func init() {
	http.HandleFunc("/probe.cgi", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		query := r.URL.Query()
		req := probeRequestFromQuery(query)

		nodes := getNodeList()
		var wg sync.WaitGroup
		wg.Add(len(nodes))
		results := make(map[string]*probeResult)
		for _, ni := range nodes {
			res := &probeResult{}
			results[ni.ID] = res
			go func(ni nodeInfo) {
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				probe(ctx, ni, req, res)
				wg.Done()
			}(ni)
		}
		wg.Wait()

		j, _ := json.Marshal(results)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(j)
	})
}
