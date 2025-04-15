#!/bin/bash
echo "plain_text mykey mysecret" > /etc/cvmfs/keys/test.repo.org.gw
cp /etc/cvmfs/gateway/repo.json.backup /etc/cvmfs/gateway/repo.json
systemctl start httpd
cvmfs_server mkfs -o root  test.repo.org
systemctl start cvmfs-gateway
