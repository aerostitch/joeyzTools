package main

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	lockCalls   int
	unlockCalls int
)

// mutexMock is used to verify that the mutex locks are properly triggered
type mutexMock struct {
	sync.Locker
}

func (m *mutexMock) Lock() {
	lockCalls++
}

func (m *mutexMock) Unlock() {
	unlockCalls++
}

func TestNewBucketCounter(t *testing.T) {
	expected := bucketCounter{
		fileMutex: &sync.Mutex{},
		fileCount: 0,
		sizeMutex: &sync.Mutex{},
		sizeCount: map[string]uint64{
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
		},
		sizeTotal:      0,
		storageMutex:   &sync.Mutex{},
		storageCount:   map[string]uint64{},
		rootMutex:      &sync.Mutex{},
		rootCount:      map[string]*bucketCounter{},
		extensionMutex: &sync.Mutex{},
		extensionCount: map[string]uint64{},
		dateMutex:      &sync.Mutex{},
		dateCount:      map[string]uint64{},
		dateRange: map[string]uint64{
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
		},
	}

	r := newBucketCounter()
	if !reflect.DeepEqual(*r, expected) {
		t.Errorf("Expecting %v got %v", expected, *r)
	}
}

func BenchmarkCountFile(b *testing.B) {
	c := newBucketCounter()
	for n := 0; n < b.N; n++ {
		c.countFile()
	}
}

func TestCountFile(t *testing.T) {
	c := newBucketCounter()
	c.fileMutex = &mutexMock{}
	lockCalls = 0
	unlockCalls = 0
	c.countFile()
	if c.fileCount != 1 {
		t.Errorf("Expecting fileCount to be 1, got %d", c.fileCount)
	}
	if lockCalls != 1 {
		t.Errorf("Expecting 1 mutex lock to be triggered. Got %d", lockCalls)
	}
	if unlockCalls != 1 {
		t.Errorf("Expecting 1 mutex unlock to be triggered. Got %d", unlockCalls)
	}
}

func BenchmarkCountSize(b *testing.B) {
	val := int64(150)
	c := newBucketCounter()
	for n := 0; n < b.N; n++ {
		c.countSize(&val)
	}
}

func TestCountSize(t *testing.T) {
	testData := []struct {
		inputSize                                                                  int64
		label                                                                      string
		initialLabelCount, expectedLabelCount, initialTotalSize, expectedTotalSize uint64
	}{
		{100, "<1KB", 90, 91, 1100, 1200},
		{104857600, "100MB-1GB", 8, 9, 55, 104857655},
	}
	for _, d := range testData {
		c := newBucketCounter()
		c.sizeCount[d.label] = d.initialLabelCount
		c.sizeTotal = d.initialTotalSize
		c.sizeMutex = &mutexMock{}
		lockCalls = 0
		unlockCalls = 0
		c.countSize(&d.inputSize)
		if c.sizeCount[d.label] != d.expectedLabelCount {
			t.Errorf("Expecting sizeCount to be %d, got %d", d.expectedLabelCount, c.sizeCount[d.label])
		}
		if c.sizeTotal != d.expectedTotalSize {
			t.Errorf("Expecting sizeTotal to be %d, got %d", d.expectedTotalSize, c.sizeTotal)
		}
		if lockCalls != 1 {
			t.Errorf("Expecting 1 mutex lock to be triggered. Got %d", lockCalls)
		}
		if unlockCalls != 1 {
			t.Errorf("Expecting 1 mutex unlock to be triggered. Got %d", unlockCalls)
		}
	}
}
func TestGetDateRange(t *testing.T) {
	tzLA, _ := time.LoadLocation("America/Los_Angeles")
	nowDate := time.Date(2018, 3, 13, 9, 30, 0, 0, tzLA)
	now5yAgo = nowDate.UTC().AddDate(-5, 0, 0)
	now4yAgo = nowDate.UTC().AddDate(-4, 0, 0)
	now3yAgo = nowDate.UTC().AddDate(-3, 0, 0)
	now2yAgo = nowDate.UTC().AddDate(-2, 0, 0)
	now1yAgo = nowDate.UTC().AddDate(-1, 0, 0)
	now9mAgo = nowDate.UTC().AddDate(0, -9, 0)
	now6mAgo = nowDate.UTC().AddDate(0, -6, 0)
	now3mAgo = nowDate.UTC().AddDate(0, -3, 0)
	now2mAgo = nowDate.UTC().AddDate(0, -2, 0)
	now1mAgo = nowDate.UTC().AddDate(0, -1, 0)

	testData := []struct {
		dateCheck time.Time
		expected  string
	}{
		{time.Date(2018, 2, 15, 0, 30, 0, 0, time.UTC), "<1 month"},
		{time.Date(2018, 1, 30, 0, 30, 0, 0, time.UTC), "1-2 month"},
		{time.Date(2018, 1, 5, 0, 30, 0, 0, time.UTC), "2-3 month"},
		{time.Date(2017, 11, 1, 0, 30, 0, 0, time.UTC), "3-6 month"},
		{time.Date(2017, 9, 10, 0, 30, 0, 0, time.UTC), "6-9 month"},
		{time.Date(2017, 6, 5, 0, 30, 0, 0, time.UTC), "9-12 month"},
		{time.Date(2016, 9, 15, 0, 30, 0, 0, time.UTC), "1-2 year"},
		{time.Date(2015, 4, 15, 0, 30, 0, 0, time.UTC), "2-3 year"},
		{time.Date(2015, 3, 13, 9, 40, 0, 0, tzLA), "2-3 year"},
		{time.Date(2015, 3, 13, 9, 20, 0, 0, tzLA), "3-4 year"},
		{time.Date(2014, 2, 15, 0, 30, 0, 0, time.UTC), "4-5 year"},
		{time.Date(2011, 1, 1, 0, 0, 0, 0, time.UTC), ">5 year"},
	}

	for _, d := range testData {
		r := getDateRange(&d.dateCheck)
		if r != d.expected {
			t.Errorf("Expecting \"%s\" got \"%s\"", d.expected, r)
		}
	}

}

func BenchmarkGetSizeRange107374182401(b *testing.B) {
	val := int64(107374182401)
	for n := 0; n < b.N; n++ {
		getSizeRange(&val)
	}
}
func BenchmarkGetSizeRange1048579(b *testing.B) {
	val := int64(1048579)
	for n := 0; n < b.N; n++ {
		getSizeRange(&val)
	}
}
func BenchmarkGetSizeRange1(b *testing.B) {
	val := int64(1)
	for n := 0; n < b.N; n++ {
		getSizeRange(&val)
	}
}

func TestGetSizeRange(t *testing.T) {
	testData := []struct {
		input    int64
		expected string
	}{
		{207374182400, "100GB+"},
		{10737418245, "10GB-100GB"},
		{1073741855, "1GB-10GB"},
		{104857600, "100MB-1GB"},
		{104857599, "10MB-100MB"},
		{1048577, "1MB-10MB"},
		{102450, "100KB-1MB"},
		{10240, "10KB-100KB"},
		{1500, "1KB-10KB"},
		{512, "<1KB"},
	}
	for _, d := range testData {
		lbl := getSizeRange(&d.input)
		if lbl != d.expected {
			t.Errorf("Expected %s, got %s for %d", d.expected, lbl, d.input)
		}
	}
}
