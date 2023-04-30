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
	"observe-geoip-util/mmdb"
	"observe-geoip-util/options"
	"observe-geoip-util/processors"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxMindEndpoint = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&suffix=tar.gz&license_key=%s"
	batchSize       = 12500
)

var (
	maxMindAPIKey    = flag.String("maxmind-apikey", "", "MaxMind API key")
	maxMindDBFile    = flag.String("maxmind-file", "", "Read database file instead of fetching from API")
	skipIPv6Networks = flag.Bool("skip-ipv6", false, "Skip IPv6 networks")
)

func main() {

	flag.Usage = func() {
		fmt.Println("usage: observe-geoip-util [global options] [subcommand] [options]")
		fmt.Println("\nglobal options:")
		fmt.Println("\t-maxmind-apikey		MaxMind API key")
		fmt.Println("\t-maxmind-file		Read database file instead of fetching from API")
		fmt.Println("\t-skip-ipv6		Skip ipv6 networks")
		fmt.Println("\nsubcommand: output-console:")
		fmt.Println("\nsubcommand: output-observe:")
		fmt.Println("\t-customer-id		Observe customer ID")
		fmt.Println("\t-ingest-token		Observe ingest token")
		fmt.Println("\nsubcommand: output-s3:")
		fmt.Println("\t-bucket			S3 bucket")
		fmt.Println("\t-access-key		AWS access key")
		fmt.Println("\t-secret-key		AWS secret key")
		fmt.Println("\t-region		        AWS region")
	}

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 || (*maxMindAPIKey == "" && *maxMindDBFile == "") {
		flag.Usage()
		os.Exit(1)
	}

	reader, closer, err := openDatabaseFile(*maxMindDBFile, *maxMindAPIKey)
	defer closer()
	if err != nil {
		log.Fatal(err)
	}

	var opts options.Options
	var processor func(string, options.Options, *workerpool.WorkerPool)

	cmd, args := args[0], args[1:]
	switch cmd {
	case "output-observe":
		fs := flag.NewFlagSet("output-observe", flag.ExitOnError)
		customer := fs.String("customer-id", "", "Observe customer ID")
		token := fs.String("ingest-token", "", "Observe ingest token")
		fs.Parse(args)

		if *customer == "" {
			log.Fatal("Missing customer ID")
		}
		if *token == "" {
			log.Fatal("Missing ingest token")
		}

		opts = options.Options{
			ObserveCustomerID:  *customer,
			ObserveIngestToken: *token,
			BatchSize:          batchSize,
		}
		processor = processors.ObserveProcessor
	case "output-s3":
		fs := flag.NewFlagSet("output-s3", flag.ExitOnError)
		bucket := fs.String("bucket", "", "S3 bucket")
		accessKey := fs.String("access-key", "", "access key")
		secretKey := fs.String("secret-key", "", "secret key")
		region := fs.String("region", "", "region")
		fs.Parse(args)

		if *bucket == "" {
			log.Fatal("Missing S3 bucket")
		}
		if *accessKey == "" {
			log.Fatal("Missing access key")
		}
		if *secretKey == "" {
			log.Fatal("Missing secret key")
		}
		if *region == "" {
			log.Fatal("Missing region")
		}

		opts = options.Options{
			S3Bucket:     *bucket,
			AWSAccessKey: *accessKey,
			AWSSecretKey: *secretKey,
			AWSRegion:    *region,
			BatchSize:    batchSize,
		}

		opts.S3Uploader = processors.NewUploader(opts)
		processor = processors.S3Processor
	case "output-console":
		opts = options.Options{}
		processor = processors.ConsoleProcessor
	default:
		flag.Usage()
		os.Exit(1)
	}

	process(reader, opts, processor)
}

func process(reader io.Reader, opts options.Options,
	processor func(string, options.Options, *workerpool.WorkerPool)) {

	data, err := readDatabaseFile(reader)
	if err != nil {
		log.Fatal(err)
	}

	pool := workerpool.New(1)
	total := processData(data, opts, processor, pool)
	log.Printf("Successfully queued %d records; waiting on completion...", total)
	pool.StopWait()
}

func processData(data []byte, opts options.Options,
	processor func(string, options.Options, *workerpool.WorkerPool), pool *workerpool.WorkerPool) int {

	database, err := maxminddb.FromBytes(data)
	if err != nil {
		log.Fatal(err)
	}

	total := 0
	buffer := make([]string, 0)
	networks := database.Networks(maxminddb.SkipAliasedNetworks)

	for networks.Next() {
		var record mmdb.MaxMindDBRecord
		subnet, err := networks.Network(&record)
		if err != nil {
			log.Fatal(err)
		}

		if *skipIPv6Networks && isIPv6(subnet.String()) {
			continue
		}

		record.Network = subnet.String()
		js, err := json.Marshal(record)
		if err != nil {
			log.Fatal(err)
		}

		buffer = append(buffer, string(js))
		if len(buffer) == opts.BatchSize {
			log.Printf("Queueing %d records for ingestion", len(buffer))
			processor(strings.Join(buffer, "\n"), opts, pool)
			buffer = buffer[:0]
		}
		total++
	}

	if len(buffer) > 0 {
		// flush remainder
		log.Printf("Queueing %d records for ingestion", len(buffer))
		processor(strings.Join(buffer, "\n"), opts, pool)
		buffer = buffer[:0]
	}

	return total
}

func isIPv6(subnet string) bool {
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatal(err)
	}
	return network.IP.To4() == nil
}

func readDatabaseFile(reader io.Reader) ([]byte, error) {

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg {
			ext := filepath.Ext(header.Name)
			if ext == ".mmdb" {
				data, err := io.ReadAll(tr)
				if err != nil {
					return nil, err
				}
				return data, nil
			}
		}
	}

	return nil, errors.New("nothing to read from tar archive")
}

func openDatabaseFile(filename, apikey string) (io.Reader, func(), error) {

	if filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			return nil, nil, err
		}

		gz, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, nil, err
		}

		return gz, func() {
			gz.Close()
			file.Close()
		}, nil
	} else {
		url := fmt.Sprintf(maxMindEndpoint, apikey)
		log.Printf("Fetching geoip database: %s\n", url)

		response, err := http.Get(url)
		if err != nil {
			return nil, nil, err
		}

		gz, err := gzip.NewReader(response.Body)
		if err != nil {
			return nil, nil, err
		}

		return gz, func() {
			gz.Close()
		}, nil
	}
}
