# HTTP benchmark

## `http_benchmark.go`

This script calls the given URL a given amount of time and returns the statistics about the http calls in a CSV format.

The statistics are much like the statistics you have with the curl commands:
  * `DNS step duration`: duration of the step that does the DNS call and only that
  * `DNS connection shared`: whether the Addrs were shared with another caller who was doing the same DNS lookup concurrently
  * `time_namelookup`: time from the start of the command until name resolution was finished
  * `Connect step duration`: duration of the step that establishes the connection, excluding the time taken to do the DNS query and get the connection from the pool
  * `time_connect`: time from the start until the remote host connection was made
  * `TLS step duration`: time it took to establish the TLS handshake if any (only that step is taken in account here, not the connection or the DNS part)
  * `Headers sent after`: time from the start until the headers of the request were sent to the server
  * `Full request sent after`: time from the start until the entire request was sent to the server
  * `time_pretransfer`: time from the start until the file transfer was about to begin
  * `First byte received after`: time from the start until the first byte is received
  * `time_starttransfer`: all pretransfer time plus the time needed to calculate the result
  * `time_total`: time for the complete operation
  * `num_connects`: number of new connections made in the transfer

Note: all the times are specified in milliseconds

Usage:
```
  -iterations uint
        Number of times to call the given URL during the benchmark (default 1)
  -url string
        URL to call and get the statistics from (default "http://example.com")
  -wait-time uint
        Number of milliseconds to wait between each call
```

Example:
```
$ go run http/http_benchmark.go -url http://www.example.com -iterations 10 -wait-time 500
DNS step duration,DNS connection shared,time_namelookup,Connect step duration,time_connect,TLS step duration,Headers sent after,Full request sent after,time_pretransfer,First byte received after,time_starttransfer,time_total,num_connects
27,false,27,15,43,0,43,43,43,141,141,141,1
23,false,23,190,214,0,214,214,214,236,236,236,1
25,false,25,57,83,0,83,83,83,108,108,108,1
36,false,36,52,89,0,89,89,89,117,117,117,1
25,false,25,43,69,0,69,69,69,96,96,96,1
28,false,28,43,71,0,71,71,71,93,93,93,1
30,false,30,153,183,0,183,183,183,205,205,205,1
21,false,21,71,92,0,93,93,93,123,123,123,1
25,false,25,66,91,0,91,91,91,109,109,110,1
24,false,24,73,98,0,98,98,98,118,118,118,1

```
