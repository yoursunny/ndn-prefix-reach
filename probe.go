package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/schema"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	_ "github.com/usnistgov/ndn-dpdk/ndn/keychain" // recognize ValidityPeriod
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

var schemaDecoder = schema.NewDecoder()

type probeRequest struct {
	NameUri     string   `schema:"name"`
	Name        ndn.Name `schema:"-"`
	AddSuffix   bool     `schema:"suffix"`
	CanBePrefix bool     `schema:"cbp"`
	MustBeFresh bool     `schema:"mbf"`
}

func probeRequestFromQuery(query url.Values) (req probeRequest) {
	schemaDecoder.Decode(&req, query)
	req.Name = ndn.ParseName(req.NameUri)
	return
}

func (req probeRequest) MakeInterest() (interest ndn.Interest) {
	if req.AddSuffix {
		interest.Name = req.Name.Append(ndn.NameComponentFrom(an.TtSequenceNumNameComponent, tlv.NNI(rand.Uint64())))
	} else {
		interest.Name = req.Name
	}
	interest.CanBePrefix = req.CanBePrefix
	interest.MustBeFresh = req.MustBeFresh
	interest.Lifetime = 3000 * time.Millisecond
	interest.HopLimit = 64
	return
}

type probeResult struct {
	OK    bool                    `json:"ok"`
	RTT   nnduration.Milliseconds `json:"rtt,omitempty"`
	Error string                  `json:"error,omitempty"`
}

func probe(ctx context.Context, ni nodeInfo, req probeRequest, res *probeResult) {
	fw, e := connect(ni)
	if e != nil {
		res.Error = e.Error()
		return
	}

	interest := req.MakeInterest()
	t0 := time.Now()
	_, e = endpoint.Consume(ctx, interest, endpoint.ConsumerOptions{
		Fw: fw,
	})
	if e == nil {
		t1 := time.Now()
		res.OK = true
		res.RTT = nnduration.Milliseconds(t1.Sub(t0).Milliseconds())
	} else {
		res.Error = e.Error()
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
		results := map[string]*probeResult{}
		for _, ni := range nodes {
			res := &probeResult{}
			results[ni.ID] = res
			go func(ni nodeInfo) {
				defer wg.Done()
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				probe(ctx, ni, req, res)
			}(ni)
		}
		wg.Wait()

		j, _ := json.Marshal(results)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(j)
	})
}
