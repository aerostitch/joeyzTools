# Get versioning statistics in Aerospike


## `aerospike_versions_statistics.go`

This script scans a set in aerospike and pulls out 2 reports from it:
 - the number of keys per generation
 - the number of keys per day of expiration of the key

Script usage example:
```
./get_versions_statistics -host localhost -namespace zombiespace -set brains -scan-percent 100 -login $myuser -password $mypwd
```

Parameters:
```
 -expirations-csv-path string
          Path to the csv report by expiration date (default "/tmp/expirations.csv")
 -generations-csv-path string
          Path to the csv report by generation (default "/tmp/generations.csv")
 -host string
          aerospike hostname or IP to connect to (default "localhost")
 -login string
          login to use to connect to aerospike when security is enabled
 -namespace string
          namespace of the aerospike set to analyze (default "default")
 -password string
          password to use to connect to aerospike when security is enabled
 -port int
          port to use to connect to aerospike (default 3000)
 -scan-percent int
          percentage of the set to scan when getting a sample (default 100)
 -set string
          aerospike set to analyze (default "test")
```

To cross-compile it in a docker for a linux 64bits (from the directory where the script is):
```
docker run --rm -v "$PWD":$(pwd) -w $(pwd) -it golang:latest /bin/bash
go get github.com/aerospike/aerospike-client-go
env GOOS=linux GOARCH=amd64 go build -v get_versions_statistics.go
```
