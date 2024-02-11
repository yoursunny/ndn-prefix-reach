package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

var (
	nodeList    atomic.Pointer[[]nodeInfo]
	errPosition = errors.New("position missing")
)

func init() {
	emptyNodeList := []nodeInfo{}
	nodeList.Store(&emptyNodeList)
}

type nodeInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Host       string     `json:"host"`
	Position   [2]float64 `json:"position"`
	Prefix     string     `json:"prefix"`
	Up         bool       `json:"up"`
	FchEnabled bool       `json:"fchEnabled"`
}

func (ni *nodeInfo) UnmarshalJSON(wire []byte) error {
	var j struct {
		ShortName    string    `json:"shortname"`
		Name         string    `json:"name"`
		Site         string    `json:"site"`
		Position     []float64 `json:"position"`
		RealPosition []float64 `json:"_real_position"`
		Prefix       string    `json:"prefix"`
		Up           bool      `json:"ndn-up"`
		FchEnabled   bool      `json:"fch-enabled"`
	}
	if e := json.Unmarshal(wire, &j); e != nil {
		return e
	}
	ni.ID = j.ShortName
	ni.Name = j.Name
	ni.Prefix = strings.TrimPrefix(j.Prefix, "ndn:")
	ni.Up = j.Up
	ni.FchEnabled = j.FchEnabled

	u, e := url.Parse(j.Site)
	if e != nil {
		return e
	}
	ni.Host = u.Hostname()

	if len(j.RealPosition) == 2 {
		copy(ni.Position[:], j.RealPosition)
	} else if len(j.Position) == 2 {
		copy(ni.Position[:], j.Position)
	} else {
		return errPosition
	}

	return nil
}

func updateNodeList() {
	time.AfterFunc(time.Duration(600+rand.Intn(60))*time.Second, updateNodeList)

	response, e := http.Get("https://testbed-status.named-data.net/testbed-nodes.json")
	if e != nil {
		log.Println("updateNodeList GET", e)
		return
	}
	body, e := io.ReadAll(response.Body)
	if e != nil {
		log.Println("updateNodeList read", e)
		return
	}

	var nodeMap map[string]nodeInfo
	if e := json.Unmarshal(body, &nodeMap); e != nil {
		log.Println("updateNodeList unmarshal", e)
		return
	}

	newNodeList := []nodeInfo{}
	for _, ni := range nodeMap {
		if ni.Host == "0.0.0.0" {
			continue
		}
		newNodeList = append(newNodeList, ni)
	}

	nodeList.Store(&newNodeList)
}

func getNodeList() []nodeInfo {
	return *nodeList.Load()
}

func init() {
	go updateNodeList()
	http.HandleFunc("/nodes.json", func(w http.ResponseWriter, r *http.Request) {
		nodes := getNodeList()
		j, _ := json.Marshal(nodes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(j)
	})
}
