// This scipt scans a set in aerospike and pulls out 2 reports from it:
//  - the number of keys per generation
//  - the number of keys per day of expiration of the key
//
// Script usage example:
// ./get_versions_statistics -host localhost -namespace zombiespace -set brains -scan-percent 100 -login $myuser -password $mypwd
//
// Parameters:
//  -expirations-csv-path string
//           Path to the csv report by expiration date (default "/tmp/expirations.csv")
//  -generations-csv-path string
//           Path to the csv report by generation (default "/tmp/generations.csv")
//  -host string
//           aerospike hostname or IP to connect to (default "localhost")
//  -login string
//           login to use to connect to aerospike when security is enabled
//  -namespace string
//           namespace of the aerospike set to analyze (default "default")
//  -password string
//           password to use to connect to aerospike when security is enabled
//  -port int
//           port to use to connect to aerospike (default 3000)
//  -scan-percent int
//           percentage of the set to scan when getting a sample (default 100)
//  -set string
//           aerospike set to analyze (default "test")
//
// To cross-compile it in a docker for a linux 64bits (from the directory where the script is):
// docker run --rm -v "$PWD":$(pwd) -w $(pwd) -it golang:latest /bin/bash
// go get github.com/aerospike/aerospike-client-go
// env GOOS=linux GOARCH=amd64 go build -v get_versions_statistics.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	as "github.com/aerospike/aerospike-client-go"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

const intermediateDateForm = "20060102"

// MapToCsv writes a csv to filepath starting with header as header line and data as data.
// If keyIsDate is set to True, the key is expected to be a date "YYYYMMDD"
// and will be written in format "YYYY-MM-DD' in the csv file.
func MapToCsv(filepath string, header string, data map[uint32]uint64, keyIsDate bool) {
	const outDateForm = "2006-01-02"
	fo, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	writer := bufio.NewWriter(fo)
	fmt.Fprintln(writer, header)
	for key, value := range data {
		if keyIsDate {
			data, err := time.Parse(intermediateDateForm, strconv.FormatUint(value, 10))
			if err != nil {
				log.Printf("Unable to convert the date back from: %f - %s\n", value, err)
			}
			fmt.Fprintln(writer, data.Format(outDateForm))
		} else {
			fmt.Fprintln(writer, key, ",", value)
		}
	}
	writer.Flush()
}

// ProcessRecord processes the record given as parameter
// meaning it updates the generations and expirations maps with the corresponding statistics.
func ProcessRecord(res *as.Result, generations map[uint32]uint64, expirations map[uint32]uint64) {
	t := time.Now()
	expDate, err := strconv.ParseUint(t.Add(time.Duration(res.Record.Expiration)*time.Second).Format(intermediateDateForm), 10, 32)
	exp_date := uint32(expDate)
	if err != nil {
		log.Printf("Unable to convert the expiration ttl to a uint date: %f - %s\n", res.Record.Expiration, err)
	}
	gen := res.Record.Generation
	if _, ok := generations[gen]; ok {
		generations[gen]++
	} else {
		generations[gen] = 1
	}
	if _, ok := expirations[exp_date]; ok {
		expirations[exp_date]++
	} else {
		expirations[exp_date] = 1
	}
}

func main() {
	var (
		host, login, pwd, namespace, set, genFile, expFile string
		port, pctScan                                      int
	)
	flag.StringVar(&host, "host", "localhost", "aerospike hostname or IP to connect to")
	flag.IntVar(&port, "port", 3000, "port to use to connect to aerospike")
	flag.StringVar(&login, "login", "", "login to use to connect to aerospike when security is enabled")
	flag.StringVar(&pwd, "password", "", "password to use to connect to aerospike when security is enabled")
	flag.StringVar(&namespace, "namespace", "default", "namespace of the aerospike set to analyze")
	flag.StringVar(&set, "set", "test", "aerospike set to analyze")
	flag.IntVar(&pctScan, "scan-percent", 100, "percentage of the set to scan when getting a sample")
	flag.StringVar(&genFile, "generations-csv-path", "/tmp/generations.csv", "Path to the csv report by generation")
	flag.StringVar(&expFile, "expirations-csv-path", "/tmp/expirations.csv", "Path to the csv report by expiration date")
	flag.Parse()

	cpolicy := as.NewClientPolicy()
	cpolicy.Timeout = 30 * time.Second
	cpolicy.IdleTimeout = 10 * time.Second
	if login != "" {
		cpolicy.User = login
		cpolicy.Password = pwd
	}

	client, err := as.NewClientWithPolicy(cpolicy, host, port)
	if err != nil {
		log.Println(err)
	}
	defer client.Close()

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.FailOnClusterChange = false
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false
	spolicy.IncludeLDT = false
	spolicy.ScanPercent = pctScan

	recs, err := client.ScanAll(spolicy, namespace, set)
	if err != nil {
		log.Println(err)
	}
	defer recs.Close()

	var generations, expirations map[uint32]uint64
	generations = make(map[uint32]uint64)
	expirations = make(map[uint32]uint64)
	var counter float64
	counter = 0

	for res := range recs.Results() {
		if res.Err != nil {
			log.Println("[ERROR] while parsing the records: ", err)
			break
		}

		ProcessRecord(res, generations, expirations)

		counter++
		if math.Mod(counter, 1000) == 0 {
			log.Printf("Processed %24.f entries\n", counter)
		}
	}

	// Writing the reports to disk
	MapToCsv(genFile, "Generation,Number of keys", generations, false)
	MapToCsv(expFile, "Expiration day,Number of keys", expirations, true)
}
