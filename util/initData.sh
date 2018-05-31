#!/bin/bash
# @Author: zealotnt
# @Date:   2018-05-31 10:26:36

echo
echo "Create testkey"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'set testkey abcdef'

echo
echo "Create testlist"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'rpush testlist 1 2 3 4'

echo
echo "Create testset"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'sadd testset 1 2 3'

echo
