# HTTP reuqests replay

## `dl_and_replay.sh`

Downloads files from s3 and make them process by `replay_fom_files.go`

## `replay_fom_files.go`

Reads files containing base64-encoded HTTP request and replay them against a
given target.
