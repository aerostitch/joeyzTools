package main

// This script replays the HTTP GET requests coming from the result of a syslog
// content listing the HTTP requests of a webserver
//
// Usage example:
// go run ./http_get_replay_fom_files.go --file-to-ingest="/tmp/*2015-07-22*.log" --header-fields "Cookie" --filter-requests "X-Forwarded-Host: www.example.com" --target-domain "replay.example.com" --verbose --rm

import (
	"bufio"
	"encoding/base64"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Defining script arguments
var (
	fileToIngest   = flag.String("file-to-ingest", "", "File to ingest. Accepts glob expressions, in which case the file will be treated 1 by 1.")
	removeFile     = flag.Bool("rm", false, "Removes the file as it is done without fatal error. Default to false.")
	maxRetries     = flag.Uint("max-retries", 3, "Defines the number of times a failing HTTP call will be retried. Default to 3.")
	targetDomain   = flag.String("target-domain", "", "DNS or IP that you want to send you HTTP requests to during the replay. No default vaule. Required field.")
	filterRequests = flag.String("filter-requests", "", "Only replay requests containing this header. Empty means all.")
	headerFields   = flag.String("header-fields", "", "Coma-separated list of HTTP headers you want to replay. Empty means all.")
	verbose        = flag.Bool("verbose", false, "Provides additional information messages.")
)

var (
	// Defines the timeout of an HTTP request
	timeoutHTTP = time.Duration(10 * time.Second)
	// The HTTP client used to execute the requests
	client = &http.Client{Timeout: timeoutHTTP}
	// Regex used to extract the last field of the syslog message (which is the
	// base64 encoded http request
	re = regexp.MustCompile("([^ ]+)$")
	// Regex used to extract the HTTP path of the request
	reget = regexp.MustCompile("GET (/\\?)*([^ ]+)")
	// Regex used to extract the headers
	reHeader = regexp.MustCompile("^(.+): (.+)$")
	// Array made from header
	headerFieldsFilter []string
)

// execRequest sends an HTTP query based on the uri m the domain and the cookie.
// It retries on errors until the retry reaches maxRetries.
// Parameters:
//  - uri is a pointer to the uri string (path not containing the domain)
//  - headers is the headers to add the the HTTP GET request
//  - retry is the current retry integer, just set it to 0
// Returns: Nothing
func execRequest(uri *string, headers *(map[string]string), retry uint) {

	if *verbose {
		log.Printf("[INFO] execRequest has been called with the following parameters: %s %v\n", *uri, *headers)
	}
	if len(*uri) > 0 {

		req, _ := http.NewRequest("GET", "http://"+(*targetDomain)+(*uri), nil)
		if *verbose {
			log.Printf("[INFO] Request to: %v", req.URL)
		}

		// Adding headers to the request
		for lbl, val := range *headers {
			if *verbose {
				log.Printf("[INFO] Adding Header: %s: %s", lbl, val)
			}
			req.Header.Add(lbl, val)
		}

		// Security to avoid the too many open files error
		req.Close = true
		resp, err := client.Do(req)
		if *verbose {
			log.Printf("%v\n%s\n", resp, err)
		}
		if err != nil {

			// Replay on errors up to maxRetries times
			if retry > *maxRetries {
				log.Printf("[ERROR] Processing of the uri %s, headers: %v failed after %d retries", *uri, *headers, retry)
				log.Fatal(err)
			} else {
				log.Printf("[ERROR] Processing of the uri %s, headers: %v failed retry # %d", *uri, *headers, retry)
				log.Println(err)
			}
			retry++
			execRequest(uri, headers, retry)
		} else {
			if resp.Body != nil {
				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}
}

// parseSyslogRequest reads a base64-encoded HTTP request and returns the
// path and headers or an error.
//
// Parameters:
//  - a pointer to the base64-encoded HTTP request
//
// Returns:
//  - a pointer to the HTTP path
//  - the list of headers found in the file
//  - any potential error encountered while parsing the HTTP request
func parseSyslogRequest(msg *string) (*string, *(map[string]string), error) {
	// Decoding the syslog message
	data, err := base64.StdEncoding.DecodeString(*msg)
	if err != nil {
		log.Println("[ERROR] ", err)
		return nil, nil, err
	}
	dec := string(data[:])

	var uri string
	headers := make(map[string]string)
	// We're only taking lines containing the defined header
	if len(*filterRequests) == 0 || strings.Contains(dec, *filterRequests) {
		for _, elt := range strings.Split(dec, "\n") {
			//log.Printf("%s\n", elt)
			// Extracting the uri path and cookie
			if reget.MatchString(elt) {
				uri = strings.Replace(reget.FindStringSubmatch(elt)[2], "/?", "?", -1)
			}
			if reHeader.MatchString(elt) {
				label := reHeader.FindStringSubmatch(elt)[1]
				value := reHeader.FindStringSubmatch(elt)[2]
				if len(headerFieldsFilter) > 0 {
					for _, field := range headerFieldsFilter {
						if field == label {
							headers[label] = value
						}
					}
				} else {
					headers[label] = value
				}
			}
		}
	}
	return &uri, &headers, nil
}

// Main function
func main() {
	flag.Parse()

	// targetDomain is a mandatory field
	if len(*targetDomain) == 0 {
		log.Fatal("The --target-domain script argument is required.")
	}

	// Splitting the headerFields if specified
	if len(*headerFields) > 0 {
		headerFieldsFilter = strings.Split(*headerFields, ",")
	}

	files, err := filepath.Glob(*fileToIngest)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if *verbose {
			log.Printf("[INFO] Replaying: %s\n", f)
		}
		fh, err2 := os.Open(f)
		if err2 != nil {
			log.Fatal(err2)
		}
		defer fh.Close()

		scanner := bufio.NewScanner(fh)
		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				str := re.FindString(line)
				uri, headers, err := parseSyslogRequest(&str)
				if err == nil {
					execRequest(uri, headers, 0)
				}
			} else {
				log.Printf("[WARNING] Encoded string not found on this line: %s\n", line)
			}

			if err := scanner.Err(); err != nil {
				if *verbose {
					log.Printf("[INFO] List of files: %v\n", files)
				}
				log.Printf("[ERROR] from the file scanner. We were processing file: %s\n", f)
				log.Fatal(err)
			}
		}
		if *removeFile {
			if *verbose {
				log.Printf("[INFO] Deleting file %s\n", f)
			}
			os.Remove(f)
		}
	}
}
