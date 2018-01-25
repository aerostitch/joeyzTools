#!/bin/bash
# Author: Joseph Herlant
# Last modification date: 2014-08-31

# Don't forget to change that to fit your own environement
HBASE_HOME=/usr/local/hbase
HADOOP_HOME=/usr/local/hadoop
PATH=$PATH:$HADOOP_HOME/bin:$HBASE_HOME/bin
HDFS_NAMENODE=localhost
HDFS_PORT=12345
HDFS_USER=$(whoami)
REF_FILE=/etc/hosts
TARGET_FILE=temporary_file.todelete
HBASE_TEMP_TABLE=anImprobableNameForATempTable

# This function checks the given return code. When RC = 0, it does nothing.
# When the RC != 0, logs the error messages and exits the script with RC 1.
# Usage: check_RC $? "Potential error message!"
check_RC()
{
  if [ $# -ne 2 ]; then
    check_RC 1 "Wrong usage of check_RC function! (requires 2 parameters)\n\
Usage example: check_RC  \$? 'Error message!'"
    return
  fi
  RC=$1
  msg=$2
  if [ $RC -ne 0 ]; then
    echo -e "\e[0;31m[ERROR]\e[m ${msg}\nReturn code was: ${RC}\nAborting..."
    exit 1
  fi
}

# First check that the user has a folder with its name at the root of hdfs.
# If not, trying to create it
hadoop fs -ls /${HDFS_USER} 2>&1 >/dev/null
if [ $? -eq 255 ]
then
  hadoop fs -mkdir /${HDFS_USER}
  check_RC $? "Unable to create /${HDFS_USER} folder in HDFS."
fi

# Then copy the file to hdfs
hadoop fs -copyFromLocal ${REF_FILE} hdfs://${HDFS_NAMENODE}:${HDFS_PORT}/${HDFS_USER}/${TARGET_FILE}
check_RC $? "Unable to copy file ${REF_FILE} to hdfs://${HDFS_NAMENODE}:${HDFS_PORT}/${HDFS_USER}/${TARGET_FILE}"
# And check that their checksum match
hadoop fs -cat /${HDFS_USER}/${TARGET_FILE} | md5sum > /tmp/hdfs_file.md5
check_RC $? "There was an error during the read of the hdfs file."
cat ${REF_FILE} | md5sum -c /tmp/hdfs_file.md5
check_RC $? "There was an error during the md5 comparison."
# Finally delete the hdfs file
hadoop fs -rm /${HDFS_USER}/${TARGET_FILE}
check_RC $? "There was an error while removing the hdfs /${HDFS_USER}/${TARGET_FILE} file."

# Then testing whether table exists or not
number_of_rows=$(echo "list '$HBASE_TEMP_TABLE'" | hbase shell  | tail -n2 | head -n1 | cut -f1,2 -d" ")
check_RC $? "There was an error while retrieving tables listing in HBase."
# If not, create it
if [ "$number_of_rows" == "0 row(s)" ]; then
  echo "create '$HBASE_TEMP_TABLE', 'testfamily'" | hbase shell
  check_RC $? "Unable to create table $HBASE_TEMP_TABLE."
fi

# Then testing batch input to hbase shell, manipulating data and finally disable and drop table.
hbase shell << __EOF__
put '$HBASE_TEMP_TABLE', 'AC/DC', 'testfamily:1st brother', 'Malcom Young'
put '$HBASE_TEMP_TABLE', 'AC/DC', 'testfamily:2nd brother', 'Angus Young'
put '$HBASE_TEMP_TABLE', 'AC/DC', 'testfamily:profession', 'Rhythm giver'
put '$HBASE_TEMP_TABLE', 'AC/DC', 'testfamily:country', 'Australia'
put '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:bigvoice', 'Steven Tyler'
put '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:goldfingers', 'Joe Perry'
put '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:profession', 'Rythm giver'
put '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:country', 'USA'
put '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:cell', 'to delete'
scan '$HBASE_TEMP_TABLE'
get '$HBASE_TEMP_TABLE', 'Aerosmith'
delete '$HBASE_TEMP_TABLE', 'Aerosmith', 'testfamily:cell'
get '$HBASE_TEMP_TABLE', 'Aerosmith'
disable '$HBASE_TEMP_TABLE'
drop '$HBASE_TEMP_TABLE'
__EOF__
check_RC $? "Unable to play with table data and drop the table after the game."

exit 0
