package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var listOnly bool

type NodeList struct {
	Items []Node `json:"items"`
}

type Node struct {
	Metadata   Metadata    `json:"metadata"`
	Status     Status      `json:"status"`
}

type Metadata struct {
	Name        string            `json:"name,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

type Status struct {
	MaxCapacity Capacity `json:"capacity"`
	Allocatable Capacity `json:"allocatable"`
	Conditions []Condition `json:"conditions"`
}


// EphemeralStorage stores a big number. We decode it as string for now...
// Memory stores a big number. We decode it as string for now...
type Capacity struct {
	CPU string `json:"cpu"`
	EphemeralStorage string `json:"ephemeral-storage"`
	Memory string `json:"memory"`
	Pods string `json:"pods"`
}

type Condition struct{
	Type    string `json:"type"`
	Status  string `json:"status"`
}

func main() {
	flag.BoolVar(&listOnly, "l", false, "List current annotations and exist")
	flag.Parse()

	expectedConditions := []Condition{
		{
			Type : "NetworkUnavailable",
			Status : "False",
		},
		{
			Type : "MemoryPressure",
			Status : "False",
		},
		{
			Type : "DiskPressure",
			Status : "False",
		},
		{
			Type : "PIDPressure",
			Status : "False",
		},
		{
			Type : "Ready",
			Status : "True",
		},
	}

	prices := []string{"0.05", "0.10", "0.20", "0.40", "0.80", "1.60"}
	resp, err := http.Get("http://127.0.0.1:8001/api/v1/nodes")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		fmt.Println("Invalid status code", resp.Status)
		os.Exit(1)
	}

	var nodes NodeList
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&nodes)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if listOnly {
		for _, node := range nodes.Items {
			price := node.Metadata.Annotations["hightower.com/cost"]
			fmt.Printf("%s %s\n", node.Metadata.Name, price)
		}
		os.Exit(0)
	}

	rand.Seed(time.Now().Unix())
	for _, node := range nodes.Items {
		// @TODO: 
		//   Request metrics
		//   To calculate price based on metrics
		//   Get metrics from --> http://localhost:8001/api/v1/nodes/<node_name>/proxy/stats/summary
		price := prices[rand.Intn(len(prices))]
		// price := "0.00"
		
		getResp, getErr := http.Get("http://127.0.0.1:8001/api/v1/nodes/" + node.Metadata.Name + "/proxy/stats/summary")
		if getErr != nil {
			fmt.Println(getErr)
			os.Exit(1)
		}
		if getResp.StatusCode != 200 {
			fmt.Println("Invalid status code", getResp.Status)
			os.Exit(1)
		}

		// @TODO: 
		//        To make structs and decoder

		// This is not O(n^2) 
		for _, expected := range expectedConditions {
			for _, nodeCondition := range node.Status.Conditions{
				if expected.Type == nodeCondition.Type {
					if expected.Status != nodeCondition.Status {
						price = "999.99"
					}
				}
			}
		}

		annotations := map[string]string{
			"hightower.com/cost": price,
		}
		patch := Node{
			Metadata{
				Annotations: annotations,	
			},
			Status{
				MaxCapacity: node.Status.MaxCapacity,
				Allocatable: node.Status.Allocatable,
				Conditions:  node.Status.Conditions,
			},
		}
		var b []byte
		body := bytes.NewBuffer(b)
		err := json.NewEncoder(body).Encode(patch)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		url := "http://127.0.0.1:8001/api/v1/nodes/" + node.Metadata.Name
		request, err := http.NewRequest("PATCH", url, body)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		request.Header.Set("Content-Type", "application/strategic-merge-patch+json")
		request.Header.Set("Accept", "application/json, */*")

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if resp.StatusCode != 200 {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("%s %s\n", node.Metadata.Name, price)
		fmt.Println(node.Status.Allocatable)
		fmt.Println(node.Status.MaxCapacity)
	}
}
