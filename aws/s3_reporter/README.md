# S3 reporter

This script generates reports about files age, extensions, size and other
attributes for one or all of your s3 buckets.

## Table of Contents

  * [Usage](#usage)
    * [Scan all your buckets](#scan-all-your-buckets)
    * [Excluding buckets](#excluding-buckets)
    * [Scan only a given list of buckets](#scan-only-a-given-list-of-buckets)
    * [Specify a path for the output report](#specify-a-path-for-the-output-report)
  * [Reports type](#reports-type)
    * [report of type summary](#report-of-type-summary)
    * [report of type details](#report-of-type-details)
    * [report of type full](#report-of-type-full)

## Usage

S3 reporter uses the Shared Config mode meaning it reads from your environent
variables and your local `~/.aws` configuration to determine the AWS credentials
to use.

Flags list (each flag has a corresponding environment variable):

```
$ ./s3_reporter -h
Usage of ./s3_reporter:
  -buckets string
        Coma-separated list of bucket to scan. If none specified, all buckets will be scanned. Environment variable: BUCKETS
  -exclude-buckets string
        Coma-separated list of bucket to exclude from the scan. Environment variable: EXCLUDE_BUCKETS
  -report-path string
        Path to the csv report to generate. Environment variable: REPORT_PATH (default "/tmp/s3.csv")
  -report-type string
        Type of report to output. Allowed values 'summary' (only size and age global report), 'details' (only details tables for each bucket), 'full' (summary + details). Environment variable: REPORT_TYPE (default "full")
```

Note: the `-report-type` will be explained in the next section.

### Scan all your buckets

S3 reporter can scan all your available repositories, for example this will scan
all your buckets and run a `full` report in `/tmp/s3.csv`:

```
./s3_reporter
```

### Excluding buckets

If you want to exclude specific buckets from the scan you can use the
`-exclude-buckets` flag. For example, this will exclude from the scan
`bigBucket1` and `bigBucket2`:

```
./s3_reporter -exclude-buckets bigBucket1,bigBucket2
```

Or using the `EXCLUDE_BUCKETS` environment variable:

```
export EXCLUDE_BUCKETS=bigBucket1,bigBucket2
./s3_reporter
```

### Scan only a given list of buckets

If you want to scan only a few buckets, you can use the `-buckets` flag. In the
example bellow, we use only the `foo` and `bar` buckets:

```
./s3_reporter -buckets foo,bar
```

Or using the `BUCKETS` environment variable:

```
export BUCKETS=foo,bar
./s3_reporter
```

### Specify a path for the output report

The `-report-path` flag is used to specify the path of the csv file you want to
save your data in. The following example shows how to scan only 1 `foo` bucket and to
output it in the file `~/reports/s3_foo.csv`:

```
./s3_reporter -buckets foo -report-path ~/reports/s3_foo.csv
```

Or using environment variables:
```
export BUCKETS=foo
export REPORT_PATH=~/reports/s3_${BUCKETS}.csv
./s3_reporter
```

## Reports type

### report of type `summary`

The summary report generates 2 tables.

The first one is about the size of the files present in your bucket.
The table will provide with the number of files present in each size ranges
defined in the header. For example:

| bucket name                              | Total number of files | Total size (GB) | <1KB   | 1KB-10KB | 10KB-100KB | 100KB-1MB | 1MB-10MB | 10MB-100MB | 100MB-1GB | 1GB-10GB | 10GB-100GB | 100GB+ |
| :--------------------------------------- | --------------------: | --------------: | -----: | -------: | ---------: | --------: | -------: | ---------: | --------: | -------: | ---------: | -----: |
| myBucket1                                | 492248                | 178.0509        | 2      | 21       | 3491       | 488730    | 4        | 0          | 0         | 0        | 0          | 0      |
| myBucket2                                | 373493                | 453.4618        | 48375  | 120143   | 122044     | 29236     | 25273    | 28422      | 0         | 0        | 0          | 0      |
| myBucket3                                | 47477                 | 95.3714         | 225    | 225      | 4659       | 20559     | 21756    | 53         | 0         | 0        | 0          | 0      |
| myBucket4                                | 1                     | 0.0000          | 0      | 1        | 0          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myBucket5                                | 0                     | 0.0000          | 0      | 0        | 0          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myBucket6                                | 485489                | 2381.2887       | 164150 | 65676    | 22735      | 1954      | 230076   | 578        | 313       | 7        | 0          | 0      |
| myBucket7                                | 27238                 | 0.1562          | 2787   | 20891    | 3560       | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myBucket8                                | 3811                  | 76.1811         | 29     | 0        | 1          | 28        | 1325     | 2428       | 0         | 0        | 0          | 0      |
| myBucket9                                | 0                     | 0.0000          | 0      | 0        | 0          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myBucket10                               | 21                    | 1.3477          | 1      | 0        | 0          | 12        | 2        | 5          | 0         | 1        | 0          | 0      |

The second table is about the age of the files present in your bucket. It helps
identify buckets with missing expiration policies or buckets that haven't been
used in a long time. It will provide the number of files in the different time
ranges shown in the header. For example:

| Bucket name                         | Total number of files | <1 month | 1-2 month | 2-3 month | 3-6 month | 6-9 month | 9-12 month | 1-2 year | 2-3 year | 3-4 year | 4-5 year | >5 year |
| :---------------------------------- | --------------------: | -------: | --------: | --------: | --------: | --------: | ---------: | -------: | -------: | -------: | -------: | ------: |
| myBucket1                           | 492248                | 45529    | 70676     | 73594     | 76417     | 77006     | 104635     | 44391    | 0        | 0        | 0        | 0       |
| myBucket2                           | 373493                | 35651    | 44148     | 46295     | 152568    | 94831     | 0          | 0        | 0        | 0        | 0        | 0       |
| myBucket3                           | 47477                 | 0        | 0         | 0         | 0         | 0         | 16811      | 30666    | 0        | 0        | 0        | 0       |
| myBucket4                           | 1                     | 0        | 0         | 0         | 0         | 1         | 0          | 0        | 0        | 0        | 0        | 0       |
| myBucket5                           | 0                     | 0        | 0         | 0         | 0         | 0         | 0          | 0        | 0        | 0        | 0        | 0       |
| myBucket6                           | 485489                | 64840    | 59943     | 52355     | 112237    | 81922     | 80127      | 34065    | 0        | 0        | 0        | 0       |
| myBucket7                           | 27238                 | 3948     | 4364      | 4139      | 11816     | 2971      | 0          | 0        | 0        | 0        | 0        | 0       |
| myBucket8                           | 3811                  | 112      | 124       | 116       | 354       | 350       | 336        | 1365     | 724      | 330      | 0        | 0       |
| myBucket9                           | 0                     | 0        | 0         | 0         | 0         | 0         | 0          | 0        | 0        | 0        | 0        | 0       |
| myBucket10                          | 21                    | 0        | 0         | 1         | 0         | 0         | 0          | 20       | 0        | 0        | 0        | 0       |




### report of type `details`

The detailed report generates 4 tables per bucket:

 * The 1st table shows the repartition of file sizes by root folder for the given bucket

| root folder for bucket myBucket99 | Total number of files | Total size (GB) | <1KB  | 1KB-10KB | 10KB-100KB | 100KB-1MB | 1MB-10MB | 10MB-100MB | 100MB-1GB | 1GB-10GB | 10GB-100GB | 100GB+ |
| :-------------------------------- | --------------------: | --------------: | ----: | -------: | ---------: | --------: | -------: | ---------: | --------: | -------: | ---------: | -----: |
| myFolder1                         | 2                     | 0.0000          | 0     | 0        | 2          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myFolder2                         | 177533                | 1278.5493       | 31230 | 9077     | 3170       | 1602      | 132310   | 142        | 2         | 0        | 0          | 0      |
| myFolder3                         | 466                   | 6.1041          | 0     | 54       | 50         | 2         | 176      | 184        | 0         | 0        | 0          | 0      |
| myFolder4                         | 2                     | 0.0000          | 2     | 0        | 0          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myFolder5                         | 10                    | 0.0003          | 2     | 2        | 6          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myFolder6                         | 21                    | 0.0026          | 0     | 0        | 10         | 11        | 0        | 0          | 0         | 0        | 0          | 0      |
| myfile.txt                        | 1                     | 0.0000          | 0     | 1        | 0          | 0         | 0        | 0          | 0         | 0        | 0          | 0      |
| myFolder7                         | 4051                  | 0.0067          | 3961  | 61       | 16         | 13        | 0        | 0          | 0         | 0        | 0          | 0      |


 * The 2nd table shows the number of files per storage class in the given bucket

| Storage class | Number of files |
| :------------ | --------------: |
| STANDARD      | 32359           |
| STANDARD_IA   | 149727          |

 * The 3rd table shows the number of files grouping by their extension (Note:
   the 1st line in the bellow example is due to files without extension)

| Extension | Number of files | 
| :-------- | --------------: | 
|           | 150840          | 
| .bin      | 200             | 
| .env      | 2               | 
| .gz       | 30509           | 
| .html     | 35              | 
| .json     | 106             | 
| .tfstate  | 11              | 
| .zip      | 383             | 

 * The 4th table shows the number of files grouped by the 1st day of the month
   it has been last modified

| Month      | Number of files | 
|------------|---------------- | 
| 2016-09-01 | 214             | 
| 2016-10-01 | 858             | 
| 2016-11-01 | 2319            | 
| 2016-12-01 | 2339            | 
| 2017-01-01 | 1561            | 
| 2017-02-01 | 1764            | 
| 2017-03-01 | 2491            | 
| 2017-04-01 | 3010            | 
| 2017-05-01 | 3025            | 
| 2017-06-01 | 3040            | 
| 2017-07-01 | 2570            | 
| 2017-08-01 | 2937            | 
| 2017-09-01 | 2668            | 
| 2017-10-01 | 3139            | 
| 2017-11-01 | 26354           | 
| 2017-12-01 | 35420           | 
| 2018-01-01 | 43220           | 
| 2018-02-01 | 30138           | 
| 2018-03-01 | 15019           | 


### report of type `full`

This is the default value of the `-report-type` flag. It adds the tables from
both the `summary` report and the tables of the `details` report in the csv
file.
