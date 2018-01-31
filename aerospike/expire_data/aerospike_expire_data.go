package main

// This script expires the records of a set that are planned to expire in less
// than 10 days and counts them.

import (
	"flag"
	as "github.com/aerospike/aerospike-client-go"
	"log"
	"math"
	"time"
)

func main() {
	var (
		host, login, pwd, namespace, set string
		port, pctScan                    int
	)
	flag.StringVar(&host, "host", "localhost", "aerospike hostname or IP to connect to")
	flag.IntVar(&port, "port", 3000, "port to use to connect to aerospike")
	flag.StringVar(&login, "login", "", "login to use to connect to aerospike when security is enabled")
	flag.StringVar(&pwd, "password", "", "password to use to connect to aerospike when security is enabled")
	flag.StringVar(&namespace, "namespace", "default", "namespace of the aerospike set to analyze")
	flag.StringVar(&set, "set", "test", "aerospike set to analyze")
	flag.IntVar(&pctScan, "scan-percent", 100, "percentage of the set to scan when getting a sample")
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

	var counter, deleteCounter float64
	counter = 0
	deleteCounter = 0

	for res := range recs.Results() {
		if res.Err != nil {
			log.Println("[ERROR] while parsing the records: ", err)
			break
		}

		if res.Record.Expiration < 864000 {
			// log.Printf("Try me: %d\n", res.Record.Expiration)
			writePolicy := as.NewWritePolicy(0, 0)
			writePolicy.Expiration = 1
			writePolicy.RecordExistsAction = 1
			client.Put(writePolicy, res.Record.Key, nil)
			deleteCounter++
		}

		counter++
		if math.Mod(counter, 1000) == 0 {
			log.Printf("Processed %24.f entries, deleted: %24.f\n", counter, deleteCounter)
		}
	}

}
