package main

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	now5yAgo = time.Now().UTC().AddDate(-5, 0, 0)
	now4yAgo = time.Now().UTC().AddDate(-4, 0, 0)
	now3yAgo = time.Now().UTC().AddDate(-3, 0, 0)
	now2yAgo = time.Now().UTC().AddDate(-2, 0, 0)
	now1yAgo = time.Now().UTC().AddDate(-1, 0, 0)
	now9mAgo = time.Now().UTC().AddDate(0, -9, 0)
	now6mAgo = time.Now().UTC().AddDate(0, -6, 0)
	now3mAgo = time.Now().UTC().AddDate(0, -3, 0)
	now2mAgo = time.Now().UTC().AddDate(0, -2, 0)
	now1mAgo = time.Now().UTC().AddDate(0, -1, 0)
)

type bucketCounter struct {
	fileMutex      sync.Locker
	fileCount      uint64
	sizeMutex      sync.Locker
	sizeCount      map[string]uint64
	sizeTotal      uint64
	storageMutex   sync.Locker
	storageCount   map[string]uint64
	rootMutex      sync.Locker
	rootCount      map[string]*bucketCounter
	extensionMutex sync.Locker
	extensionCount map[string]uint64
	dateMutex      sync.Locker
	dateCount      map[string]uint64
	dateRange      map[string]uint64
}

// newBucketCounter initialize a new bucketCounter with the required fields
// initialized
func newBucketCounter() *bucketCounter {
	c := bucketCounter{}
	c.initStats()
	return &c
}

// countFile increments the file counter
func (c *bucketCounter) countFile() {
	c.fileMutex.Lock()
	c.fileCount++
	c.fileMutex.Unlock()
}

// countSize increments the different size counters
func (c *bucketCounter) countSize(keySize int64) {
	k := getSizeRange(keySize)
	c.sizeMutex.Lock()
	c.sizeCount[k]++
	c.sizeTotal += uint64(keySize)
	c.sizeMutex.Unlock()
}

// countDateSummary increments the date summary counters
func (c *bucketCounter) countDateSummary(keyDate time.Time) {
	k := getDateRange(keyDate)
	incrementUint64(c.dateMutex, c.dateRange, k)
}

// incrementUint64 increments a map[string]uint64 using a mutex
func incrementUint64(m sync.Locker, ctr map[string]uint64, key string) {
	m.Lock()
	// Relies on the fact that a null value for an uint64 is 0
	ctr[key]++
	m.Unlock()
}

// increment increments a *bucketCounter
func (c *bucketCounter) increment(size int64, storageClass, extension, root string, lastModified time.Time, recurse bool) {
	lastMod := fmt.Sprintf("%d-%02d-01", lastModified.Year(), lastModified.Month())
	c.countFile()
	c.countSize(size)
	c.countDateSummary(lastModified)
	incrementUint64(c.storageMutex, c.storageCount, storageClass)
	incrementUint64(c.extensionMutex, c.extensionCount, extension)
	incrementUint64(c.dateMutex, c.dateCount, lastMod)
	if recurse {
		c.rootMutex.Lock()
		ctr, ok := c.rootCount[root]
		if !ok {
			ctr = newBucketCounter()
		}
		ctr.rootCount[root] = ctr
		c.rootMutex.Unlock()
		ctr.increment(size, storageClass, extension, root, lastModified, false)
	}
}

// initStats initialize the statistics of a bucketCounter
func (c *bucketCounter) initStats() {
	c.fileMutex = &sync.Mutex{}
	c.fileCount = 0
	c.sizeMutex = &sync.Mutex{}
	c.sizeCount = map[string]uint64{
		"100GB+":     0,
		"10GB-100GB": 0,
		"1GB-10GB":   0,
		"100MB-1GB":  0,
		"10MB-100MB": 0,
		"1MB-10MB":   0,
		"100KB-1MB":  0,
		"10KB-100KB": 0,
		"1KB-10KB":   0,
		"<1KB":       0,
	}
	c.sizeTotal = 0
	c.storageMutex = &sync.Mutex{}
	c.storageCount = make(map[string]uint64)
	c.rootMutex = &sync.Mutex{}
	c.rootCount = make(map[string]*bucketCounter)
	c.extensionMutex = &sync.Mutex{}
	c.extensionCount = make(map[string]uint64)
	c.dateMutex = &sync.Mutex{}
	c.dateCount = make(map[string]uint64)
	c.dateRange = map[string]uint64{
		"<1 month":   0,
		"1-2 month":  0,
		"2-3 month":  0,
		"3-6 month":  0,
		"6-9 month":  0,
		"9-12 month": 0,
		"1-2 year":   0,
		"2-3 year":   0,
		"3-4 year":   0,
		"4-5 year":   0,
		">5 year":    0,
	}

}

// getDateRange returns the key label corresponding to the range the given date is in
func getDateRange(keyDate time.Time) string {
	switch {
	case keyDate.Before(now5yAgo):
		return ">5 year"
	case keyDate.Before(now4yAgo):
		return "4-5 year"
	case keyDate.Before(now3yAgo):
		return "3-4 year"
	case keyDate.Before(now2yAgo):
		return "2-3 year"
	case keyDate.Before(now1yAgo):
		return "1-2 year"
	case keyDate.Before(now9mAgo):
		return "9-12 month"
	case keyDate.Before(now6mAgo):
		return "6-9 month"
	case keyDate.Before(now3mAgo):
		return "3-6 month"
	case keyDate.Before(now2mAgo):
		return "2-3 month"
	case keyDate.Before(now1mAgo):
		return "1-2 month"
	}
	return "<1 month"
}

// getSizeRange returns the key label corresponding to the range the given size is in
func getSizeRange(keySize int64) string {
	switch {
	case keySize >= 107374182400:
		return "100GB+"
	case keySize >= 10737418240:
		return "10GB-100GB"
	case keySize >= 1073741824:
		return "1GB-10GB"
	case keySize >= 104857600:
		return "100MB-1GB"
	case keySize >= 10485760:
		return "10MB-100MB"
	case keySize >= 1048576:
		return "1MB-10MB"
	case keySize >= 102400:
		return "100KB-1MB"
	case keySize >= 10240:
		return "10KB-100KB"
	case keySize >= 1024:
		return "1KB-10KB"
	}
	return "<1KB"
}

// reportDateSummary provides the reports on date-related statistics for a map[string]*bucketCounter
func reportDateSummary(csvWriter *csv.Writer, ctr map[string]*bucketCounter) error {
	if err := csvWriter.Write([]string{"Repartition of file ages by buckets"}); err != nil {
		return err
	}
	header := []string{"Bucket name", "Total number of files", "<1 month", "1-2 month", "2-3 month", "3-6 month", "6-9 month", "9-12 month", "1-2 year", "2-3 year", "3-4 year", "4-5 year", ">5 year"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for k, v := range ctr {
		arr := []string{
			k,
			strconv.FormatUint(v.fileCount, 10),
			strconv.FormatUint(v.dateRange["<1 month"], 10),
			strconv.FormatUint(v.dateRange["1-2 month"], 10),
			strconv.FormatUint(v.dateRange["2-3 month"], 10),
			strconv.FormatUint(v.dateRange["3-6 month"], 10),
			strconv.FormatUint(v.dateRange["6-9 month"], 10),
			strconv.FormatUint(v.dateRange["9-12 month"], 10),
			strconv.FormatUint(v.dateRange["1-2 year"], 10),
			strconv.FormatUint(v.dateRange["2-3 year"], 10),
			strconv.FormatUint(v.dateRange["3-4 year"], 10),
			strconv.FormatUint(v.dateRange["4-5 year"], 10),
			strconv.FormatUint(v.dateRange[">5 year"], 10),
		}
		if err := csvWriter.Write(arr); err != nil {
			return err
		}
	}
	if err := csvWriter.Write(nil); err != nil {
		return err
	}
	csvWriter.Flush()
	return nil
}

// reportSizing provides the reports on size-related statistics for a map[string]*bucketCounter
func reportSizing(csvWriter *csv.Writer, ctr map[string]*bucketCounter, byColumn string) error {
	if err := csvWriter.Write([]string{fmt.Sprintf("Repartition of file sizes by %s", byColumn)}); err != nil {
		return err
	}
	header := []string{byColumn, "Total number of files", "Total size (GB)", "<1KB", "1KB-10KB", "10KB-100KB", "100KB-1MB", "1MB-10MB", "10MB-100MB", "100MB-1GB", "1GB-10GB", "10GB-100GB", "100GB+"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for k, v := range ctr {
		arr := []string{
			k,
			strconv.FormatUint(v.fileCount, 10),
			strconv.FormatFloat((float64(v.sizeTotal) / 1024.0 / 1024.0 / 1024.0), 'f', 4, 64),
			strconv.FormatUint(v.sizeCount["<1KB"], 10),
			strconv.FormatUint(v.sizeCount["1KB-10KB"], 10),
			strconv.FormatUint(v.sizeCount["10KB-100KB"], 10),
			strconv.FormatUint(v.sizeCount["100KB-1MB"], 10),
			strconv.FormatUint(v.sizeCount["1MB-10MB"], 10),
			strconv.FormatUint(v.sizeCount["10MB-100MB"], 10),
			strconv.FormatUint(v.sizeCount["100MB-1GB"], 10),
			strconv.FormatUint(v.sizeCount["1GB-10GB"], 10),
			strconv.FormatUint(v.sizeCount["10GB-100GB"], 10),
			strconv.FormatUint(v.sizeCount["100GB+"], 10),
		}
		if err := csvWriter.Write(arr); err != nil {
			return err
		}
	}
	if err := csvWriter.Write(nil); err != nil {
		return err
	}
	csvWriter.Flush()
	return nil
}

// reportByRoot Reports the repartition of files by root folder for a given bucket
func reportByRoot(csvWriter *csv.Writer, bucket string, ctr *bucketCounter) error {
	var err error
	if ctr != nil && len(ctr.rootCount) > 0 {
		err = reportSizing(csvWriter, ctr.rootCount, fmt.Sprintf("root folder for bucket %s", bucket))
	}
	return err
}

// reportUint64 exports in a csv format a map[string]uint64 under a give title
// and set of headers
func reportUint64(csvWriter *csv.Writer, ctr map[string]uint64, title string, headers []string) error {
	var err error
	if len(ctr) > 0 {
		if err = csvWriter.Write([]string{title}); err != nil {
			return err
		}
		if err = csvWriter.Write(headers); err != nil {
			return err
		}
		// To store the keys in slice in sorted order
		var keys []string
		for k := range ctr {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err = csvWriter.Write([]string{k, strconv.FormatUint(ctr[k], 10)}); err != nil {
				return err
			}
		}
		if err = csvWriter.Write(nil); err != nil {
			return err
		}
		csvWriter.Flush()
	}
	return err
}
