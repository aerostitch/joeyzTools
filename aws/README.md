# AWS-related scripts

## `iam_get_user_from_access_key`

This script either list all keys on an account
or list the user corresponding to the given key


## `ec2_security_group_reports`

This script list the security groups with ports opened to the world for IPv4.


## `cloudwatch_metrics_density`

This script reports the number of metrics in cloudwatch for each namespace /
metric / metric dimension that can be pulled over the last 2 hours. It helps
tracking the density of the metrics and this way find who stores at least 1
metric per second. This is needed when you try to track down the spend in the
monthly report for the PutMetrics operation.


## `elb_log_analyzer`

The script in `elb_log_analyzer` is a tool to parse AWS Elastic LoadBalancer
access logs and push them to a MySQL database.

## `s3_reporter`

This script generates reports about files age, extensions, size and other attributes for one or all of your s3 buckets.
