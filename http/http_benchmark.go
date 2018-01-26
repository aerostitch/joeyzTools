package main

/*
This script calls the given URL a given amount of time and returns the statistics about the http calls in a CSV format.
The statistics are much like the statistics you have with the curl commands:
  * DNS step duration: duration of the step that does the DNS call and only that
	* DNS connection shared: whether the Addrs were shared with another caller who was doing the same DNS lookup concurrently
	* time_namelookup: time from the start of the command until name resolution was finished
	* Connect step duration: duration of the step that establishes the connection, excluding the time taken to do the DNS query and get the connection from the pool
	* time_connect: time from the start until the remote host connection was made
	* TLS step duration: time it took to establish the TLS handshake if any (only that step is taken in account here, not the connection or the DNS part)
	* Headers sent after: time from the start until the headers of the request were sent to the server
	* Full request sent after: time from the start until the entire request was sent to the server
	* time_pretransfer: time from the start until the file transfer was about to begin
	* First byte received after: time from the start until the first byte is received
	* time_starttransfer: all pretransfer time plus the time needed to calculate the result
	* time_total: time for the complete operation
	* num_connects: number of new connections made in the transfer

Note: all the times are specified in milliseconds

*/

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

// TODO: get headers like X-Cache and CDN and host
func processURL(targetURL string) {
	var (
		dnsStart, dnsDone, connectStart, connectDone, tlsStart, tlsDone, headersSent, requestSent, firstByte, requestExecuted time.Time
		dnsShared                                                                                                             bool
	)

	numConnections := 0
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsShared = dnsInfo.Coalesced // Coalesced is whether the Addrs were shared with another caller who was doing the same DNS lookup concurrently
			dnsDone = time.Now()
		},
		ConnectStart: func(_, _ string) { numConnections++; connectStart = time.Now() },
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Fatalf("unable to connect to host %v: %v", addr, err)
			}
			connectDone = time.Now()
		},
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { tlsDone = time.Now() },
		WroteHeaders:         func() { headersSent = time.Now() },
		WroteRequest:         func(_ httptrace.WroteRequestInfo) { requestSent = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	_, err = http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Fatal(err)
	}
	requestExecuted = time.Now()

	queryStart := dnsStart
	// In case the IP is called directly
	if dnsStart.IsZero() {
		queryStart = connectStart
	}
	fmt.Printf("%d,%t,%d,", dnsDone.Sub(dnsStart)/time.Millisecond, dnsShared, dnsDone.Sub(dnsStart)/time.Millisecond)
	fmt.Printf("%d,%d,", connectDone.Sub(connectStart)/time.Millisecond, connectDone.Sub(queryStart)/time.Millisecond)
	fmt.Printf("%d,", tlsDone.Sub(tlsStart)/time.Millisecond)
	fmt.Printf("%d,%d,%d,", headersSent.Sub(queryStart)/time.Millisecond, requestSent.Sub(queryStart)/time.Millisecond, requestSent.Sub(queryStart)/time.Millisecond)
	fmt.Printf("%d,%d,%d,", firstByte.Sub(queryStart)/time.Millisecond, firstByte.Sub(queryStart)/time.Millisecond, requestExecuted.Sub(queryStart)/time.Millisecond)
	fmt.Printf("%d\n", numConnections)
}

func main() {
	var (
		urlInput          string
		iterations, pause uint
	)
	flag.StringVar(&urlInput, "url", "http://example.com", "URL to call and get the statistics from")
	flag.UintVar(&iterations, "iterations", 1, "Number of times to call the given URL during the benchmark")
	flag.UintVar(&pause, "wait-time", 0, "Number of milliseconds to wait between each call")
	flag.Parse()

	// Just making sure the url passed is a correct url
	urlObj, err := url.Parse(urlInput)
	if err != nil {
		log.Fatal(err)
	}
	checkedURL := urlObj.String()
	fmt.Println("DNS step duration,DNS connection shared,time_namelookup,Connect step duration,time_connect,TLS step duration,Headers sent after,Full request sent after,time_pretransfer,First byte received after,time_starttransfer,time_total,num_connects")
	for ; iterations > 0; iterations-- {
		processURL(checkedURL)
		if pause > 0 {
			time.Sleep(time.Duration(pause) * time.Millisecond)
		}
	}
}
