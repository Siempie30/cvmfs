#!/bin/bash

USE_ATTACH_CMD=false
while getopts "A" opt; do
  case $opt in
    A)
      USE_ATTACH_CMD=true
      ;;
    *)
      echo "Usage: $0 [-A] <repo_name>"
      exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

# Ensure a repository name is provided
if [ $# -ne 1 ]; then
  echo "Usage: $0 [-E] <repo_name>"
  exit 1
fi

REPO_NAME=$1

echo "plain_text mykey mysecret" > /etc/cvmfs/keys/${REPO_NAME}.gw
systemctl start httpd

# Configure and set up repository
echo "CVMFS_S3_ACCESS_KEY=minioadmin
CVMFS_S3_SECRET_KEY=minioadmin123
CVMFS_S3_HOST=cvmfs-s3
CVMFS_S3_PORT=9000
CVMFS_S3_BUCKET=stratum0bucket
CVMFS_S3_DNS_BUCKETS=false
CVMFS_S3_USE_HTTPS=false" > /etc/cvmfs/s3.conf

CREATE_CMD="cvmfs_server"
if [ "$USE_ATTACH_CMD" = true ]; then
  CREATE_CMD="$CREATE_CMD attach -o root http://cvmfs-s3:9000/stratum0bucket /etc/cvmfs/s3.conf"
else
  CREATE_CMD="$CREATE_CMD mkfs -o root -s /etc/cvmfs/s3.conf -w http://cvmfs-s3:9000/stratum0bucket "
fi
CREATE_CMD="$CREATE_CMD $REPO_NAME"

# Execute the create command
$CREATE_CMD