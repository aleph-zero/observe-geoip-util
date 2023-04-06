# observe-geoip-util

Utility to download a Maxmind GeoIP database and upload to Observe (https://www.observeinc.com). Can optionally read `.mmdb` files from the command line. You will need to register for a free API key from Maxmind (https://www.maxmind.com).

go build -o observe-geoip-util main.go 

```
./observe-geoip-util -h
    -maxmind-api-key      MaxMind API key
    -maxmind-file         Read database file instead of fetching from API
    -observe-customer-id  Observe customer ID
    -observe-ingest-token Observe ingest token
    -output-json          Print results to JSON
```
