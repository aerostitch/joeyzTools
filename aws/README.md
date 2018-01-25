# AWS-related scripts

## `aws_iam_get_user_from_access_key.go`

This script either list all keys on an account
or list the user corresponding to the given key

Parameters:
* `-filter-key` (default: ""): Aws access key to look for. If not provided, will list all of them.

Usage example:
```
go run aws_iam_get_user_from_access_key.go -filter-key AKIABCDEFGHIJKLMNOPQ
```

## `aws_ec2_security_groups_opened.go`

This script list the security groups with ports opened to the world for IPv4.
