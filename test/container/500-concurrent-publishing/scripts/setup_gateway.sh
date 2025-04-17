#!/bin/bash

# Install minio client
curl -o /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x /usr/local/bin/mc

# Configure and create the bucket
mc alias set local http://cvmfs-s3:9000 minioadmin minioadmin123
mc mb local/mybucket
mc anonymous set public local/mybucket

echo "plain_text mykey mysecret" > /etc/cvmfs/keys/test.repo.org.gw
cp /etc/cvmfs/gateway/repo.json.backup /etc/cvmfs/gateway/repo.json
systemctl start httpd

echo "CVMFS_S3_ACCESS_KEY=minioadmin
CVMFS_S3_SECRET_KEY=minioadmin123
CVMFS_S3_HOST=cvmfs-s3
CVMFS_S3_PORT=9000
CVMFS_S3_BUCKET=mybucket
CVMFS_S3_DNS_BUCKETS=false
CVMFS_S3_USE_HTTPS=false" > /etc/cvmfs/s3.conf

cvmfs_server mkfs -s /etc/cvmfs/s3.conf -w http://cvmfs-s3:9000/mybucket -o root  test.repo.org
systemctl start cvmfs-gateway
