# Aerospike-related scripts


## `versions_statistics`

This script scans a set in aerospike and pulls out 2 reports from it:
 - the number of keys per generation
 - the number of keys per day of expiration of the key


## `expire_data`

This script expires the records of a set that are planned to expire in less
than 10 days and counts them.
