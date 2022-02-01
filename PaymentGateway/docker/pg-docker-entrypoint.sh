#!/bin/bash
if [[ "${no_conf}" != "1" ]]; then
  if [[ "${PP_ENV}" != "prod" ]]; then
    export UseTestApi=true
  else
    export UseTestApi=false
  fi
  cat /opt/torplus/config.json.tmpl | envsubst > /opt/torplus/config.json
fi
mkdir -p /opt/torplus/logs
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
    /opt/torplus/payment-gateway | mark "Server is ready!" ".pg_ready" &> /opt/torplus/logs/payment.log
else 
    exec "$@" | mark "Server is ready!" ".pg_ready"
fi
