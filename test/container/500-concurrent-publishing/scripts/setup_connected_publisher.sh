#!/bin/bash

# Check if the gateway URL is provided as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 <CVMFS_GATEWAY_URL>"
  exit 1
fi

CVMFS_GATEWAY_URL=$1

yum -y install jq sqlite

FQRN=test.repo.org
CVMFS_STRATUM0_URL=http://cvmfs-s3:9000/mybucket
CVMFS_SERVER_DEBUG=3 cvmfs_server mkfs -w $CVMFS_STRATUM0_URL/$FQRN \
                         -u gw,/srv/cvmfs/$FQRN/data/txn,$CVMFS_GATEWAY_URL:4929/api/v1 \
                         -k /etc/cvmfs/keys -o `whoami` $FQRN

