package processors

import (
	"errors"
	"fmt"
	"github.com/gammazero/workerpool"
	"log"
	"net/http"
	"observe-geoip-util/options"
	"strings"
	"time"
)

const (
	observeEndpoint = "https://%s.collect.observeinc.com/v1/http/maxmind-geoip"
)

func ObserveProcessor(payload string, options options.Options, pool *workerpool.WorkerPool) {

	pool.Submit(func() {
		submit(payload, options.ObserveCustomerID, options.ObserveIngestToken)
	})
}

func submit(data string, customerId string, token string) {

	url := fmt.Sprintf(observeEndpoint, customerId)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{
		Timeout: 60 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal(errors.New(res.Status))
	}
}
