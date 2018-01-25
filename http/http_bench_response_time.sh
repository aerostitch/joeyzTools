#!/bin/bash
# This script's purpose is to test the response time on a url
# by doing a simple curl loop (no pause, so a bit hammering).
# It returns the several curl metrics in a csv file for further analysis.
#
TO="localhost"
OUTFILE="$(date +%Y%m%d%H%M)_curl_test_from_$(hostname)_to_${TO}.csv"
CURLURI="http://${TO}/crossdomain.xml"
CURLFORMAT='\n%{time_namelookup},%{time_connect},%{time_appconnect},%{time_pretransfer},%{time_redirect},%{time_starttransfer},%{time_total},%{num_connects},%{num_redirects}'
ITERATIONS=100000
 
# Header
echo -e "$(date +%F): Testing url ${CURLURI} from $(hostname)" > $OUTFILE
echo -e "Date,time_namelookup,time_connect,time_appconnect,time_pretransfer,time_redirect,time_starttransfer,time_total,num_connects,num_redirects" >> $OUTFILE
 
i=0
while [ $i -lt $ITERATIONS ]
do
    curl -w "${CURLFORMAT}" -o /dev/null -s $CURLURI | xargs echo -e "$(date +%H:%M:%S.%6N)," >> $OUTFILE
    i=$[$i+1]
done
# vim: set nowrap:
