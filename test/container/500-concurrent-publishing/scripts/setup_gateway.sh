#!/bin/bash

# Configure and create the bucket
mc alias set local http://cvmfs-s3:9000 minioadmin minioadmin123
mc mb local/mybucket
mc anonymous set public local/mybucket

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

echo "{
  \"repos\": [
    {
      \"repoName\": \"test.repo.org\",
      \"gateways\": [
        \"http://cvmfs-gw1:4929/api/v1\"
      ]
    }
  ]
}" > /etc/cvmfs/gateway/token_ring.json

echo "{
    \"max_lease_time\" : 21,
    \"port\" : 4929,
    \"num_receivers\": 1,
    \"receiver_path\": \"/usr/bin/cvmfs_receiver\",
    \"log_level\" : \"info\",
    \"log_timestamps\" : false,
    \"work_dir\": \"/var/lib/cvmfs-gateway\",
	\"gw_lease_acquisition_time\": 20
}" > /etc/cvmfs/gateway/user.json

cvmfs_server mkfs -s /etc/cvmfs/s3.conf -w http://cvmfs-s3:9000/mybucket -o root  test.repo.org
systemctl start cvmfs-gateway
