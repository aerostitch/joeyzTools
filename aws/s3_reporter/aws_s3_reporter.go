package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/gobike/envflag"
)

var report map[string]*bucketCounter
var reportMutex sync.Locker

// Profiling vars
var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

// flags
var (
	reportPath     = flag.String("report-path", "/tmp/s3.csv", "Path to the csv report to generate. Environment variable: REPORT_PATH")
	bucketsList    = flag.String("buckets", "", "Coma-separated list of bucket to scan. If none specified, all buckets will be scanned. Environment variable: BUCKETS")
	bucketsExclude = flag.String("exclude-buckets", "", "Coma-separated list of bucket to exclude from the scan. Environment variable: EXCLUDE_BUCKETS")
	reportType     = flag.String("report-type", "full", "Type of report to output. Allowed values 'summary' (only size and age global report), 'details' (only details tables for each bucket), 'full' (summary + details). Environment variable: REPORT_TYPE")
)

// getBucketsList returns the full list of buckets
func getBucketsList(svc s3iface.S3API) ([]*string, error) {
	buckets := []*string{}
	result, err := svc.ListBuckets(&s3.ListBucketsInput{})
	for _, bucket := range result.Buckets {
		buckets = append(buckets, bucket.Name)
	}
	return buckets, err
}

// getBucketRegion returns the normalized region of a given bucket
func getBucketRegion(svc s3iface.S3API, bucketName *string) (string, error) {
	var loc string
	location, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: bucketName})
	if location.LocationConstraint != nil {
		loc = *location.LocationConstraint
	}
	return s3.NormalizeBucketLocation(loc), err
}

// bucketWorker takes care of listing the objects pages and putting them in the
// page channel
func bucketWorker(sess client.ConfigProvider, sessionRegion string, svc s3iface.S3API, buckets chan *string, wg *sync.WaitGroup, pageChan chan *s3.ListObjectsV2Output) {
	for b := range buckets {
		log.Printf("%d buckets left in the queue", len(buckets))
		loc, err := getBucketRegion(svc, b)
		if err != nil {
			log.Printf("Error while retrieving the bucket %s location: %s\n", *b, err)
			continue
		}
		log.Printf("Bucket: %s, Location: %s\n", *b, loc)
		localSvc := svc
		// Makes sure we are in the right region and avoid stuffs like:
		// AuthorizationHeaderMalformed: The authorization header is malformed; the region 'us-east-1' is wrong
		if loc != sessionRegion {
			localSvc = s3.New(sess, aws.NewConfig().WithRegion(loc))
		}
		getBucketObjects(localSvc, b, pageChan)
	}
	wg.Done()
}

// getBucketObjects gets the list of objects in a bucket
func getBucketObjects(svc s3iface.S3API, bucketName *string, pageChan chan *s3.ListObjectsV2Output) {
	encodingType := "url"
	params := s3.ListObjectsV2Input{Bucket: bucketName, EncodingType: &encodingType}
	chanWarn := int(float64(cap(pageChan)) * 0.95) // threshold after which we slow down the feed to the channel
	err := svc.ListObjectsV2Pages(&params,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			if len(pageChan) > chanWarn {
				log.Println("Page channel is soon at capacity, slowing down the worker")
				time.Sleep(1 * time.Second)
			}
			pageChan <- page
			return !lastPage
		})
	if err != nil {
		log.Fatalf("ListObjectsV2Pages returned: %s", err)
	}
}

// processPage gets the statistics for each page of objects provided by the
// channel
func processPage(pageChan chan *s3.ListObjectsV2Output, wg *sync.WaitGroup) {
	for page := range pageChan {
		log.Printf("1 page of %d objects fetched for bucket %s", len(page.Contents), *page.Name)
		for _, obj := range page.Contents {
			getObjectStats(page.Name, obj)
		}
	}
	wg.Done()
}

// getObjectStats collects the statistics of an objects in the report structure
func getObjectStats(bucketName *string, obj *s3.Object) {
	lastChar := (*obj.Key)[len(*obj.Key)-1:]
	// Skips folders
	if lastChar != "/" {
		ext := path.Ext(*obj.Key)
		lastMod := (*obj.LastModified).UTC()
		root := (strings.Split(*obj.Key, "/"))[0]
		reportMutex.Lock()
		currentReport := report[*bucketName]
		reportMutex.Unlock()
		currentReport.increment(*obj.Size, *obj.StorageClass, ext, root, lastMod, true)
	}
}

// reportCsv export the current state of the report to a csv file
func reportCsv(filePath, reportType string) {
	f, errF := os.Create(filePath)
	if errF != nil {
		log.Fatalf("os.Create returned: %s", errF)
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)
	if reportType == "summary" || reportType == "full" {
		if err := reportSizing(csvWriter, report, "bucket name"); err != nil {
			log.Fatalf("reportSizing returned: %s", err)
		}

		if err := reportDateSummary(csvWriter, report); err != nil {
			log.Fatalf("reportDateSummary returned: %s", err)
		}
	}

	if reportType == "details" || reportType == "full" {
		for bucket, stats := range report {
			if err := reportByRoot(csvWriter, bucket, stats); err != nil {
				log.Fatalf("reportByRoot returned: %s", err)
			}

			if err := reportUint64(csvWriter, stats.storageCount, fmt.Sprintf("Repartition of files for bucket %s by storage class", bucket), []string{"Storage class", "Number of files"}); err != nil {
				log.Fatalf("Storage class reportUint64 returned: %s", err)
			}
			if err := reportUint64(csvWriter, stats.extensionCount, fmt.Sprintf("Repartition of files for bucket %s by extension", bucket), []string{"Extension", "Number of files"}); err != nil {
				log.Fatalf("Extension reportUint64 returned: %s", err)
			}
			if err := reportUint64(csvWriter, stats.dateCount, fmt.Sprintf("Repartition of files for bucket %s by month", bucket), []string{"Month", "Number of files"}); err != nil {
				log.Fatalf("Monthly reportUint64 returned: %s", err)
			}
		}
	}
	csvWriter.Flush()
}

func main() {
	envflag.Parse()

	if *reportType != "summary" && *reportType != "details" && *reportType != "full" {
		log.Fatal("Incorrect report-type specified. Allowed values:\n - summary: only size and age global report\n - details: only details tables for each bucket\n - full: summary + details")
	}

	// PROFILING CPU BLOCK INIT
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	// PROFILING CPU BLOCK END

	if report == nil {
		report = make(map[string]*bucketCounter)
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := s3.New(sess)
	var buckets []*string
	if len(*bucketsList) <= 0 {
		var err error
		if buckets, err = getBucketsList(svc); err != nil {
			log.Fatalf("Error while retrieving the buckets list: %s\n", err)
		}
	} else {
		for _, b := range strings.Split(*bucketsList, ",") {
			bucket := b
			buckets = append(buckets, &bucket)
		}
	}

	reportMutex = &sync.Mutex{}
	var wg, wgBucket sync.WaitGroup
	pageChan := make(chan *s3.ListObjectsV2Output, 100)
	bucketsChan := make(chan *string, 1000)
	// Setup a worker group
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go processPage(pageChan, &wg)
	}
	for i := 0; i < 8; i++ {
		wgBucket.Add(1)
		go bucketWorker(sess, *sess.Config.Region, svc, bucketsChan, &wgBucket, pageChan)
	}

	skipBuckets := strings.Split(*bucketsExclude, ",")
BUCKETS_LOOP:
	for _, b := range buckets {
		for _, skip := range skipBuckets {
			if *b == skip {
				continue BUCKETS_LOOP
			}
		}
		if _, ok := report[*b]; !ok {
			reportMutex.Lock()
			report[*b] = newBucketCounter()
			reportMutex.Unlock()
		}
		bucketsChan <- b
	}
	close(bucketsChan)

	wgBucket.Wait()
	close(pageChan)
	wg.Wait()

	reportCsv(*reportPath, *reportType)

	// MEMORY PROFILING BLOCK INIT
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
	// MEMORY PROFILING BLOCK END
}
