package main

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

type bucketCounter struct {
	fileMutex      *sync.Mutex
	fileCount      uint64
	sizeMutex      *sync.Mutex
	sizeCount      map[string]uint64
	sizeTotal      uint64
	storageMutex   *sync.Mutex
	storageCount   map[string]uint64
	rootMutex      *sync.Mutex
	rootCount      map[string]*bucketCounter
	extensionMutex *sync.Mutex
	extensionCount map[string]uint64
	dateMutex      *sync.Mutex
	dateCount      map[string]uint64
}

// newBucketCounter initialize a new bucketCounter with the required fields
// initialized
func newBucketCounter() *bucketCounter {
	c := bucketCounter{}
	initStats(&c)
	return &c
}

// countFile increments the file counter
func (c *bucketCounter) countFile() {
	c.fileMutex.Lock()
	c.fileCount++
	c.fileMutex.Unlock()
}

// countSize increments the different size counters
func (c *bucketCounter) countSize(keySize *int64) {
	k := getSizeRange(keySize)
	c.sizeMutex.Lock()
	c.sizeCount[k]++
	c.sizeTotal += uint64(*keySize)
	c.sizeMutex.Unlock()
}

// incrementUint64 increments a map[string]uint64 using a mutex
func incrementUint64(m *sync.Mutex, ctr map[string]uint64, key *string) {
	m.Lock()
	if _, ok := ctr[*key]; !ok {
		ctr[*key] = 1
	} else {
		ctr[*key]++
	}
	m.Unlock()
}

// increment increments a *bucketCounter
func increment(ctr *bucketCounter, size *int64, storageClass, extension, lastModified, root *string, recurse bool) {
	ctr.countFile()
	ctr.countSize(size)
	incrementUint64(ctr.storageMutex, ctr.storageCount, storageClass)
	incrementUint64(ctr.extensionMutex, ctr.extensionCount, extension)
	incrementUint64(ctr.dateMutex, ctr.dateCount, lastModified)
	if recurse {
		ctr.rootMutex.Lock()
		c, ok := ctr.rootCount[*root]
		if !ok {
			c = newBucketCounter()
		}
		ctr.rootCount[*root] = c
		ctr.rootMutex.Unlock()
		increment(c, size, storageClass, extension, lastModified, root, false)
	}
}

// initStats initialize the statistics of a bucketCounter
func initStats(c *bucketCounter) {
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

}

// getSizeRange returns the key label corresponding to the range the given size is in
func getSizeRange(keySize *int64) string {
	switch {
	case *keySize >= 107374182400:
		return "100GB+"
	case *keySize >= 10737418240:
		return "10GB-100GB"
	case *keySize >= 1073741824:
		return "1GB-10GB"
	case *keySize >= 104857600:
		return "100MB-1GB"
	case *keySize >= 10485760:
		return "10MB-100MB"
	case *keySize >= 1048576:
		return "1MB-10MB"
	case *keySize >= 102400:
		return "100KB-1MB"
	case *keySize >= 10240:
		return "10KB-100KB"
	case *keySize >= 1024:
		return "1KB-10KB"
	}
	return "<1KB"
}

// reportSizing provides the reports on size-related statistics for a map[string]*bucketCounter
func reportSizing(csvWriter *csv.Writer, ctr map[string]*bucketCounter) error {
	for k, v := range ctr {
		arr := []string{
			k,
			strconv.FormatUint(v.fileCount, 10),
			strconv.FormatUint(v.sizeTotal, 10),
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
		if err = csvWriter.Write([]string{fmt.Sprintf("Repartition of file sizes for bucket %s by root folder", bucket)}); err != nil {
			return err
		}
		rootHeader := []string{"Root folder", "Total number of files", "Total size", "<1KB", "1KB-10KB", "10KB-100KB", "100KB-1MB", "1MB-10MB", "10MB-100MB", "100MB-1GB", "1GB-10GB", "10GB-100GB", "100GB+"}
		if err = csvWriter.Write(rootHeader); err != nil {
			return err
		}
		err = reportSizing(csvWriter, ctr.rootCount)
	}
	return err
}

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
		if err := csvWriter.Write(nil); err != nil {
			return err
		}
		csvWriter.Flush()
	}
	return err
}
