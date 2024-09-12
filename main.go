package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	headscaleApiUrl := requireEnv("HEADSCALE_API_URL")
	headscaleApiKey := requireEnv("HEADSCALE_API_KEY")
	listenAddr := requireEnv("LISTEN_ADDR", ":8080")

	err := http.ListenAndServe(listenAddr, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		if ctx == nil {
			var cancel func()
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
		}
		response, err := HeadscaleList(context.Background(), headscaleApiUrl, headscaleApiKey)
		if err != nil {
			log.Println(err.Error())
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		discoveryTargets := ParseDiscoveryTargets(response)
		data, err := json.Marshal(discoveryTargets)
		if err != nil {
			log.Println(err.Error())
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Add("Content-Type", "application/json")
		writer.WriteHeader(200)
		_, _ = writer.Write(data)
		log.Println("success response with targets len: ", len(discoveryTargets))
	}))
	if err != nil {
		log.Fatalf("cant start server: %s", err.Error())
	}
}

func requireEnv(name string, defaults ...string) string {
	value := os.Getenv(name)
	if value != "" {
		return value
	}
	for _, d := range defaults {
		if d != "" {
			return d
		}
	}
	log.Fatalf("env %s is not set", name)
	panic("")
}

type DiscoveryTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func ParseDiscoveryTargets(list *ListMachineResponse) []*DiscoveryTarget {
	result := []*DiscoveryTarget{}
	for _, machine := range list.Machines {
		if len(machine.IpAddresses) < 1 {
			continue
		}
		ip := machine.IpAddresses[0]
		nodeName := machine.GivenName
		allTags := uniq(slices.Concat(machine.ForcedTags, machine.ValidTags, machine.InvalidTags))
		for _, tag := range allTags {
			tag = strings.TrimPrefix(tag, "tag:")
			if !strings.HasPrefix(tag, "scrape_") {
				continue
			}
			tag = strings.TrimPrefix(tag, "scrape_")
			portStr, appName, found := strings.Cut(tag, "_")
			if !found {
				continue
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}
			result = append(result, &DiscoveryTarget{
				Targets: []string{fmt.Sprintf("%s:%d", ip, port)},
				Labels: map[string]string{
					"node_name": nodeName,
					"app":       appName,
				},
			})
		}
	}
	return result
}

func uniq(list []string) []string {
	m := map[string]bool{}
	for _, item := range list {
		m[item] = true
	}
	result := make([]string, 0, len(list))
	for item := range m {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func HeadscaleList(ctx context.Context, url string, apiKey string) (*ListMachineResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url+listMachinePath, nil)
	if err != nil {
		return nil, fmt.Errorf("cant create request: %w", err)
	}

	request.Header.Add("Authorization", "Bearer "+apiKey)
	request.Header.Add("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cant make http call: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cant read body: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response status code: %d; body: %s", response.StatusCode, string(data))
	}
	parsedResponse := &ListMachineResponse{}
	if err := json.Unmarshal(data, parsedResponse); err != nil {
		return nil, fmt.Errorf("cant parse response: %w; body: %s", err, string(data))
	}
	return parsedResponse, nil
}

const listMachinePath = "/api/v1/machine"

type ListMachineResponse struct {
	Machines []struct {
		Id          string   `json:"id"`
		MachineKey  string   `json:"machineKey"`
		NodeKey     string   `json:"nodeKey"`
		DiscoKey    string   `json:"discoKey"`
		IpAddresses []string `json:"ipAddresses"`
		Name        string   `json:"name"`
		User        struct {
			Id        string    `json:"id"`
			Name      string    `json:"name"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"user"`
		LastSeen             time.Time `json:"lastSeen"`
		LastSuccessfulUpdate time.Time `json:"lastSuccessfulUpdate"`
		Expiry               time.Time `json:"expiry"`
		PreAuthKey           struct {
			User       string    `json:"user"`
			Id         string    `json:"id"`
			Key        string    `json:"key"`
			Reusable   bool      `json:"reusable"`
			Ephemeral  bool      `json:"ephemeral"`
			Used       bool      `json:"used"`
			Expiration time.Time `json:"expiration"`
			CreatedAt  time.Time `json:"createdAt"`
			AclTags    []string  `json:"aclTags"`
		} `json:"preAuthKey"`
		CreatedAt      time.Time `json:"createdAt"`
		RegisterMethod string    `json:"registerMethod"`
		ForcedTags     []string  `json:"forcedTags"`
		InvalidTags    []string  `json:"invalidTags"`
		ValidTags      []string  `json:"validTags"`
		GivenName      string    `json:"givenName"`
		Online         bool      `json:"online"`
	} `json:"machines"`
}
