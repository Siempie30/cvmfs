#!/bin/bash

# Check if the gateway URL and repo name are provided as arguments
if [ $# -lt 2 ]; then
  echo "Usage: $0 <CVMFS_GATEWAY_URL> <REPO_NAME>"
  exit 1
fi

CVMFS_GATEWAY_URL=$1
FQRN=$2

yum -y install jq sqlite

CVMFS_STRATUM0_URL=http://cvmfs-s3:9000/mybucket
CVMFS_SERVER_DEBUG=3 cvmfs_server mkfs -w $CVMFS_STRATUM0_URL/$FQRN \
                         -u gw,/srv/cvmfs/$FQRN/data/txn,$CVMFS_GATEWAY_URL:4929/api/v1 \
                         -k /etc/cvmfs/keys -o `whoami` $FQRN

