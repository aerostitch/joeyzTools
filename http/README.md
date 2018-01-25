# HTTP-related scripts

## `dl_and_replay.sh`

Downloads files from s3 and make them process by replay_fom_files.go

## `replay_fom_files.go`

Reads files containing base64-encoded HTTP request and replay them against a
given target.

## `http_bench_response_time.sh`

This script's purpose is to test the response time on a url
by doing a simple curl loop (no pause, so a bit hammering).
It returns the several curl metrics in a csv file for further analysis.
