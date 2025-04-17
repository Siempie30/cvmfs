#!/bin/bash
#jsudo yum install -y https://cvmrepo.s3.cern.ch/cvmrepo/yum/cvmfs-release-latest.noarch.rpm
#sudo yum install -y cvmfs cvmfs-server
yum -y install jq sqlite

FQRN=test.repo.org
CVMFS_GATEWAY_URL=http://cvmfs-gw1
CVMFS_STRATUM0_URL=http://cvmfs-s3:9000/mybucket
CVMFS_SERVER_DEBUG=3 cvmfs_server mkfs -w $CVMFS_STRATUM0_URL/$FQRN \
                         -u gw,/srv/cvmfs/$FQRN/data/txn,$CVMFS_GATEWAY_URL:4929/api/v1 \
                         -k /etc/cvmfs/keys -o `whoami` $FQRN

curl -X POST --data "{\"repo\":\"$FQRN\"}" $CVMFS_GATEWAY_URL:4929/api/v1/token-ring
cvmfs_server transaction
cvmfs_server publish

