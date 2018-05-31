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
  -d 'sadd testset x y z'

echo
echo "Create testset1"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'sadd testset1 a 1 2 3'

echo
echo "Create testset2"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'sadd testset2 a 1 4 5'

echo
echo "Create testset3"
curl -X POST \
  http://localhost:3000/ \
  -H 'cache-control: no-cache' \
  -d 'sadd testset3 a 1 6 7'

echo
