#!/bin/bash

# Check for the -E option
ENABLE_E_FLAG=false
while getopts "E" opt; do
  case $opt in
    E)
      ENABLE_E_FLAG=true
      ;;
    *)
      echo "Usage: $0 [-E]"
      exit 1
      ;;
  esac
done

echo "plain_text mykey mysecret" > /etc/cvmfs/keys/test.repo.org.gw
cp /etc/cvmfs/gateway/repo.json.backup /etc/cvmfs/gateway/repo.json
systemctl start httpd

# Configure and set up repository
echo "CVMFS_S3_ACCESS_KEY=minioadmin
CVMFS_S3_SECRET_KEY=minioadmin123
CVMFS_S3_HOST=cvmfs-s3
CVMFS_S3_PORT=9000
CVMFS_S3_BUCKET=mybucket
CVMFS_S3_DNS_BUCKETS=false
CVMFS_S3_USE_HTTPS=false" > /etc/cvmfs/s3.conf

# Add the -E flag if the option is enabled
MKFS_CMD="cvmfs_server mkfs -s /etc/cvmfs/s3.conf -w http://cvmfs-s3:9000/mybucket -o root"
if [ "$ENABLE_E_FLAG" = true ]; then
  MKFS_CMD="$MKFS_CMD -E"
fi
MKFS_CMD="$MKFS_CMD test.repo.org"

# Execute the mkfs command
$MKFS_CMD

systemctl start cvmfs-gateway
