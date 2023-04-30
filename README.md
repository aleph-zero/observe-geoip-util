# observe-geoip-util

Utility to download a Maxmind GeoIP database and upload to Observe (https://www.observeinc.com). Can optionally read `.mmdb` files from the command line. You will need to register for a free API key from Maxmind (https://www.maxmind.com).

### Build from source:
```go build -o observe-geoip-util main.go```

### Usage
```
usage: observe-geoip-util [global options] [subcommand] [options]

global options:
	-maxmind-apikey		MaxMind API key
	-maxmind-file		Read database file instead of fetching from API
	-skip-ipv6		Skip ipv6 networks

subcommand: output-console:

subcommand: output-observe:
	-customer-id		Observe customer ID
	-ingest-token		Observe ingest token

subcommand: output-s3:
	-bucket			S3 bucket
	-access-key		AWS access key
	-secret-key		AWS secret key
	-region		        AWS region
```

### Example usage for Observe upload:
```
./observe-geoip-util 
    -skip-ipv6 
    -maxmind-apikey <MAXMIND_KEY> 
    output-observe 
    -customer-id <OBSERVE_CUSTOMER_ID>
    -ingest-token <OBSERVE_INGEST_TOKEN>
```

### Example usage for S3 upload:
```
./observe-geoip-util 
    -skip-ipv6 
    -maxmind-file ./data/GeoLite2-City.tar.gz 
    output-s3 
    -bucket     <AWS_BUCKET>
    -access-key <AWS_ACCESS_KEY>
    -secret-key <AWS_SECRET_KEY>
    -region     <AWS_REGION>
```
