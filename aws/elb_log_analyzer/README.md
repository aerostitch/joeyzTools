# AWS ELB access logs analyzer

## `aws_ec2_elb_log_analyzer.go`

This tool analyzes access logs for classic ELB access logs (from local files or
from files stored in an s3 bucket), pushes them inside a MySQL/MariaDB
database and generate a csv that summarizes your access logs usage (optionally).

Easy way to get a DB:
```
docker run --name some-mariadb -e MYSQL_ROOT_PASSWORD=my-secret-pw -e MYSQL_DATABASE=accesslogs -p 3306:3306 -d mariadb:latest
```

To bulk load your local files in this case (without generating the standard reports):
```
DB_NAME=accesslogs
TBL=bla
go run aws_elb_log_analyzer.go  -db-host "tcp(172.17.0.2)" \
                                -db-name ${DB_NAME} \
                                -db-user root \
                                -db-pwd my-secret-pw \
                                -db-table ${TBL} \
                                -recursive \
                                -file-path /tmp/${TBL}
```


Bulk loading from S3 and generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
BUCKET_NAME=my-elb-logs-bucket
go run aws_elb_log_analyzer.go  -db-host "tcp(172.17.0.2)" \
                                -db-name ${DB_NAME} \
                                -db-user root \
                                -db-pwd my-secret-pw \
                                -db-table ${TBL} \
                                -s3-bucket ${BUCKET_NAME} \
                                -s3-path ${TBL}/AWSLogs
```

Bulk loading from local files and generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
go run aws_elb_log_analyzer.go  -db-host "tcp(172.17.0.2)" \
                                -db-name ${DB_NAME} \
                                -db-user root \
                                -db-pwd my-secret-pw \
                                -db-table ${TBL} \
                                -recursive \
                                -file-path /tmp/${TBL} \
                                -report-path /tmp/${TBL}_summary.csv
```

Only generate the standard report:
```
DB_NAME=accesslogs
TBL=bla
go run aws_elb_log_analyzer.go  -db-host "tcp(172.17.0.2)" \
                                -db-name ${DB_NAME} \
                                -db-user root \
                                -db-pwd my-secret-pw \
                                -db-table ${TBL} \
                                -report-path /tmp/${TBL}_summary.csv
```

Note that you can also go into your DB and generate your own custom reports...

Custom reports examples based on the imported data (here we exclude the calls from Pingdom and stuffs that we now are script kiddies playing around):
 * By day and IP: `select year, month, day, sourceIP, count(*) as nbrcalls from bla group by year, month, day, sourceIP order by nbrcalls;`
 * By uri: `select SUBSTRING_INDEX(uri, '?', 1), count(*) as nbrcalls from bla group by SUBSTRING_INDEX(uri, '?', 1) order by nbrcalls;`
 * By userAgent: `select SUBSTRING_INDEX(userAgent, ' (', 1), count(*) as nbrcalls from bla group by SUBSTRING_INDEX(userAgent, ' (', 1) order by nbrcalls;`
 * A bit of filtering: `select year, month, day, hour, SUBSTRING_INDEX(userAgent, ' (', 1) as agent, SUBSTRING_INDEX(uri, '?', 1) as uri, count(*) as nbrcalls from bla where userAgent not like 'Pingdom%' and userAgent != 'ZmEu' group by year, month, day, hour, SUBSTRING_INDEX(userAgent, ' (', 1), SUBSTRING_INDEX(uri, '?', 1) order by year, month, day, hour, nbrcalls;`

