#!/bin/bash
# peak RSS (MB) of server alone and of server + every descendant (exiftool, vipsthumbnail, ffmpeg), every 100 ms
srv=$1; out=$2; max=0; maxsrv=0
tree() { echo $1; for c in $(ps -o pid= --ppid $1); do tree $c; done; }
while kill -0 $srv 2>/dev/null; do
  tot=$(ps -o rss= -p $(tree $srv | paste -sd,) 2>/dev/null | awk '{s+=$1} END {print int(s/1024)}')
  s=$(ps -o rss= -p $srv | awk '{print int($1/1024)}')
  [ "$tot" -gt "$max" ] && max=$tot; [ "$s" -gt "$maxsrv" ] && maxsrv=$s
  echo "peak_total_mb=$max peak_server_mb=$maxsrv" > $out; sleep 0.1
done
