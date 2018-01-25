#!/bin/bash
#set -x
AUTH_TOKEN="<PUT_YOURS_HERE>"
V=latest

exec_qubole_query () {
  query=$1
  if [ -z "$1" ]
  then
    echo -e "[ERROR] No query sent as parameter.\n"
  else
    echo -e "[DEBUG] Running query: \"$1\".\n"
  fi
  curl -X POST -H "X-AUTH-TOKEN:$AUTH_TOKEN" -H "Content-Type: application/json" -H "Accept: application/json" \
    -d "{
  \"query\":\"${query}\"
}"\
  "https://api.qubole.com/api/${V}/commands/"
}

for day in $(seq -w 22 24)
do
  DATE=2015-04-${day}
  Q1="INSERT OVERWRITE TABLE my_new_table Partition (logdate = '${DATE}' ) SELECT * FROM my_table WHERE my_date = '${DATE}';"
  exec_qubole_query "$Q1"
  sleep 600

  echo -e '\n\n*******************************************\n\n'

done
