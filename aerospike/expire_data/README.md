# Expire datas in Aerospike

## `aerospike_expire_data.go`

This script expires the records of a set that are planned to expire in less
than 10 days and counts them.

Usage:
```
  -host string
        aerospike hostname or IP to connect to (default "localhost")
  -login string
        login to use to connect to aerospike when security is enabled
  -namespace string
        namespace of the aerospike set to analyze (default "default")
  -password string
        password to use to connect to aerospike when security is enabled
  -port int
        port to use to connect to aerospike (default 3000)
  -scan-percent int
        percentage of the set to scan when getting a sample (default 100)
  -set string
        aerospike set to analyze (default "test")
```
