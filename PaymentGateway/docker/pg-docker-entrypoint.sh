#!/bin/bash
if [[ "${no_conf}" != "1" ]]; then

  cat /opt/paidpiper/config.json.tmpl | envsubst > /opt/paidpiper/config.json
fi
function mark {
  match=$1
  file=$2
  mark=1
  while read -r data; do
    echo $data
    if [[ $data == *"$match"* ]]; then 
      if [[ "$mark" == "1" ]]; then 
        echo "done" >> $file
        mark=0
      fi
    fi
  done
}
if [ $# -eq 0 ]
then
    /opt/paidpiper/payment-gateway | mark "Server is ready!" ".pgready"
else 
    exec "$@" | mark "Server is ready!" ".pg_ready"
fi
