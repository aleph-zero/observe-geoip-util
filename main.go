package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gammazero/workerpool"
	"github.com/oschwald/maxminddb-golang"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxMindEndpoint = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&suffix=tar.gz&license_key=%s"
	observeEndpoint = "https://%s.collect.observeinc.com/v1/http/geoip?vendor=maxmind"
	batchSize       = 12500
	workerPoolSize  = 1
)

type MaxMindDBRecord struct {
	City struct {
		GeonameID uint64            `maxminddb:"geoname_id"`
		Names     map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`

	Continent struct {
		Code  string            `maxminddb:"code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`

	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		IsEU    bool              `maxminddb:"is_in_european_union"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`

	Location struct {
		Latitude       float32 `maxminddb:"latitude"`
		Longitude      float32 `maxminddb:"longitude"`
		AccuracyRadius int     `maxminddb:"accuracy_radius"`
		TimeZone       string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`

	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`

	Network string
}

func main() {

	customer := flag.String("observe-customer-id", "", "Observe customer ID")
	token := flag.String("observe-ingest-token", "", "Observe ingest token")
	apiKey := flag.String("maxmind-api-key", "", "MaxMind API key")
	output := flag.Bool("output-json", false, "Print results to JSON")
	skipv6 := flag.Bool("skip-ipv6", false, "Skip IPv6 networks")
	filename := flag.String("maxmind-file", "", "Read database file instead of fetching from API")

	flag.Parse()

	if *customer == "" {
		log.Fatal("Missing customer ID")
	}
	if *token == "" {
		log.Fatal("Missing ingest token")
	}

	log.Printf("Ingesting to Observe endpoint: %s\n", fmt.Sprintf(observeEndpoint, *customer))
	log.Printf("Skip IPv6: %t\n", *skipv6)

	var reader io.Reader

	if *filename != "" {
		file, err := os.Open(*filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		fmt.Printf("Reading geoip database: %s\n", *filename)

		gz, err := gzip.NewReader(file)
		if err != nil {
			log.Fatal(err)
		}
		defer gz.Close()
		reader = gz
	} else {
		if *apiKey == "" {
			log.Fatal("Missing API key")
		}

		url := fmt.Sprintf(maxMindEndpoint, *apiKey)
		fmt.Printf("Fetching geoip database: %s\n", url)

		response, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		gz, err := gzip.NewReader(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		defer gz.Close()
		reader = gz
	}

	process(reader, *customer, *token, *output, *skipv6)
}

func process(r io.Reader, customer string, token string, output bool, skipv6 bool) {

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if header.Typeflag == tar.TypeReg {
			extension := filepath.Ext(header.Name)
			if extension == ".mmdb" {
				data, err := io.ReadAll(tr)
				if err != nil {
					log.Fatal(err)
				}
				mmdb(data, customer, token, output, skipv6)
			}
		}
	}
}

func mmdb(data []byte, customer string, token string, output bool, skipv6 bool) {

	mmdb, err := maxminddb.FromBytes(data)
	if err != nil {
		log.Fatal(err)
	}

	var index = 0
	var batch []string
	var totalRecords = 0
	var batchesSubmitted = 0
	var batchesReturned = 0

	pool := workerpool.New(workerPoolSize)

	networks := mmdb.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {

		if index == 0 {
			batch = make([]string, batchSize)
		}

		var record MaxMindDBRecord

		subnet, err := networks.Network(&record)
		if err != nil {
			log.Fatal(err)
		}

		if skipv6 {
			_, ipv4Net, err := net.ParseCIDR(subnet.String())
			if err != nil {
				log.Fatal(err)
			}
			if ipv4Net.IP.To4() == nil {
				continue
			}
		}

		record.Network = subnet.String()

		js, err := json.Marshal(record)
		if err != nil {
			log.Fatal(err)
		}

		payload := string(js)
		if output {
			fmt.Println(payload)
		}

		batch[index] = strings.Clone(payload)
		index++
		totalRecords++

		if index == batchSize {
			data := strings.Join(batch, "\n")
			batchesSubmitted++
			i := index
			t := totalRecords

			pool.Submit(func() {
				log.Printf("Ingesting %d/%d records.\n", i, t)
				_, err := observe(data, customer, token)
				if err != nil {
					log.Fatal(fmt.Sprintf("Ingestion failed [%s] for payload: \n%s\n", err, data))
				}
				batchesReturned++
			})

			index = 0
		}
	}
	if index > 0 {
		data := strings.Join(batch, "\n")
		batchesSubmitted++
		i := index
		t := totalRecords

		pool.Submit(func() {
			log.Printf("Ingesting %d/%d records.\n", i, t)
			_, err := observe(data, customer, token)
			if err != nil {
				log.Fatal(fmt.Sprintf("Ingestion failed [%s] for payload: \n%s\n", err, data))
			}
			batchesReturned++

		})
		index = 0
	}

	log.Printf("All records queued for ingestion. Waiting on completion...")
	pool.StopWait()
	log.Printf("Ingestion complete. Total records: %d\n", totalRecords)
	log.Printf("Batches submitted: %d\n", batchesSubmitted)
	log.Printf("Batches returned: %d\n", batchesReturned)
}

func observe(payload string, customer string, token string) (int, error) {

	url := fmt.Sprintf(observeEndpoint, customer)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{
		Timeout: 60 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		return 0, errors.New(res.Status)
	}

	return res.StatusCode, nil
}

/*
	// INTERNAL STRUCTURE OF MAXMIND DB RECORDS
   map[
       city:map[geoname_id:3177363 names:map[de:Ercolano en:Ercolano fr:Ercolano pt-BR:Ercolano ru:Геркуланум]]
       continent:map[code:EU geoname_id:6255148 names:map[de:Europa en:Europe es:Europa fr:Europe ja:ヨーロッパ pt-BR:Europa ru:Европа zh-CN:欧洲]]
       country:map[geoname_id:3175395 is_in_european_union:true iso_code:IT names:map[de:Italien en:Italy es:Italia fr:Italie ja:イタリア共和国 pt-BR:Itália ru:Италия zh-CN:意大利]]
       location:map[accuracy_radius:10 latitude:40.8112 longitude:14.3528 time_zone:Europe/Rome]
       postal:map[code:80056] registered_country:map[geoname_id:3175395 is_in_european_union:true iso_code:IT names:map[de:Italien en:Italy es:Italia fr:Italie ja:イタリア共和国 pt-BR:Itália ru:Италия zh-CN:意大利]]
       subdivisions:[map[geoname_id:3181042 iso_code:72 names:map[de:Kampanien en:Campania es:Campania fr:Campanie ja:カンパニア州 pt-BR:Campânia ru:Кампания zh-CN:坎帕尼亚]] map[geoname_id:3172391 iso_code:NA names:map[de:Neapel en:Naples es:Napoles fr:Naples pt-BR:Nápoles]]]]
*/
