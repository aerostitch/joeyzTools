#!/bin/bash
# This script ingests file from s3 and leave the file on the hard drive if
# processed with errors. The current filter is set to ingest several days of
# 2015-07

DL_FOLDER=${HOME}/worklog/$(date '+%Y%m%d')_fastly/
BUCKET_NAME=fastly-logs
SCRIPT_PATH=$(dirname "${BASH_SOURCE[0]}")
FORWARDED_HOST=foo.example.com
TARGET_HOST=bar.example.com

mkdir -p $DL_FOLDER

source ~/.venv/bin/activate

for i in $(seq -w 13 19) ; do
  FILES_FILTERING="2015-07-"$i

  for sf in $(aws s3 ls "s3://${BUCKET_NAME}/" | sed 's/^[^ ]* *[^ ]* *[0-9]* *\(.*\)/\1/g' | grep "${FILES_FILTERING}"); do
    aws s3 cp "s3://${BUCKET_NAME}/${sf}" ${DL_FOLDER}
    echo "Decompressing ${DL_FOLDER}${sf}"
    gunzip -f "${DL_FOLDER}${sf}"
    echo "Replaying ${DL_FOLDER}${sf%*.gz}"
    go run ${SCRIPT_PATH}/replay_fom_files.go --file-to-ingest="${sf%*.gz}" --header-fields "Cookie" --filter-requests "X-Forwarded-Host: ${FORWARDED_HOST}" --target-domain ${TARGET_HOST} --verbose --rm > "${sf%*.gz}.ingest" 2>&1
    if [[ $? -ne 0 ]] ; then
      echo "Error, see: ${DL_FOLDER}${sf%*.gz}.ingest"
    else
      rm "${DL_FOLDER}${sf%*.gz}"
    fi
  done
done
