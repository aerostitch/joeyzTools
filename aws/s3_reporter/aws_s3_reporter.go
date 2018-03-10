package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/gobike/envflag"
)

var report map[string]*bucketCounter

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

// getBucketObjects gets the list of objects in a bucket
func getBucketObjects(svc s3iface.S3API, bucketName *string) {
	if _, ok := report[*bucketName]; !ok {
		report[*bucketName] = newBucketCounter()
	}
	encodingType := "url"
	params := s3.ListObjectsV2Input{Bucket: bucketName, EncodingType: &encodingType}
	err := svc.ListObjectsV2Pages(&params,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, obj := range page.Contents {
				getObjectStats(bucketName, obj)
			}
			return !lastPage
		})
	if err != nil {
		log.Fatalf("ListObjectsV2Pages returned: %s", err)
	}
}

// getObjectStats collects the statistics of an objects in the report structure
func getObjectStats(bucketName *string, obj *s3.Object) {
	lastChar := (*obj.Key)[len(*obj.Key)-1:]
	// Skips folders
	if lastChar != "/" {
		ext := path.Ext(*obj.Key)
		lastMod := (*obj.LastModified).UTC()
		mod := fmt.Sprintf("%d-%d-01", lastMod.Year(), lastMod.Month())
		root := (strings.Split(*obj.Key, "/"))[0]
		increment(report[*bucketName], obj.Size, obj.StorageClass, &ext, &mod, &root, true)
	}
}

// reportCsv export the current state of the report to a csv file
func reportCsv(filePath string) {
	f, errF := os.Create(filePath)
	if errF != nil {
		log.Fatalf("os.Create returned: %s", errF)
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)
	if err := csvWriter.Write([]string{"Repartition of file sizes by buckets"}); err != nil {
		log.Fatal("csvWriter.Write returned: %s", err)
	}
	header := []string{"Bucket name", "Total number of files", "Total size", "<1KB", "1KB-10KB", "10KB-100KB", "100KB-1MB", "1MB-10MB", "10MB-100MB", "100MB-1GB", "1GB-10GB", "10GB-100GB", "100GB+"}
	if err := csvWriter.Write(header); err != nil {
		log.Fatal("csvWriter.Write returned: %s", err)
	}
	if err := reportSizing(csvWriter, report); err != nil {
		log.Fatal("reportSizing returned: %s", err)
	}

	for bucket, stats := range report {
		if err := reportByRoot(csvWriter, bucket, stats); err != nil {
			log.Fatal("reportByRoot returned: %s", err)
		}

		if err := reportUint64(csvWriter, stats.storageCount, fmt.Sprintf("Repartition of files for bucket %s by storage class", bucket), []string{"Storage class", "Number of files"}); err != nil {
			log.Fatal("Storage class reportUint64 returned: %s", err)
		}
		if err := reportUint64(csvWriter, stats.extensionCount, fmt.Sprintf("Repartition of files for bucket %s by extension", bucket), []string{"Extension", "Number of files"}); err != nil {
			log.Fatal("Extension reportUint64 returned: %s", err)
		}
		if err := reportUint64(csvWriter, stats.dateCount, fmt.Sprintf("Repartition of files for bucket %s by month", bucket), []string{"Month", "Number of files"}); err != nil {
			log.Fatal("Monthly reportUint64 returned: %s", err)
		}
	}
	csvWriter.Flush()
}

func main() {
	var reportPath, bucketFilter string
	flag.StringVar(&reportPath, "report-path", "/tmp/s3.csv", "Path to the csv report to generate. Environment variable: REPORT_PATH")
	flag.StringVar(&bucketFilter, "bucket", "", "Bucket to scan. If none specified, all buckets will be scanned. Environment variable: BUCKET")
	envflag.Parse()

	if report == nil {
		report = make(map[string]*bucketCounter)
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := s3.New(sess)
	buckets := []*string{&bucketFilter}
	if len(bucketFilter) <= 0 {
		var err error
		if buckets, err = getBucketsList(svc); err != nil {
			log.Fatalf("Error while retrieving the buckets list: %s\n", err)
		}
	}
	for _, b := range buckets {
		loc, err := getBucketRegion(svc, b)
		if err != nil {
			log.Fatalf("Error while retrieving the bucket %s location: %s\n", *b, err)
		}
		log.Printf("Bucket: %s, Location: %s\n", *b, loc)
		localSvc := svc
		// Makes sure we are in the right region and avoid stuffs like:
		// AuthorizationHeaderMalformed: The authorization header is malformed; the region 'us-east-1' is wrong
		if loc != *sess.Config.Region {
			localSvc = s3.New(sess, aws.NewConfig().WithRegion(loc))
		}
		getBucketObjects(localSvc, b)
	}
	reportCsv(reportPath)
}
