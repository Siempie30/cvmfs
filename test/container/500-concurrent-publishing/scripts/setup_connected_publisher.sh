#!/bin/bash

set -e

usage() {
    echo "Usage: $0 -G <gateway url> -F <repo name> [-M]"
    exit 1
}

# Default values
MKFS_OPTS=""

# Parse arguments
while getopts "G:F:M" opt; do
    case ${opt} in
        G)
            cvmfs_gateway_url="$OPTARG"
            ;;
        F)
            fqrn="$OPTARG"
            ;;
        M)
            MKFS_OPTS="-M"
            ;;
        *)
            usage
            ;;
    esac
done

# Check required arguments
if [ -z "$cvmfs_gateway_url" ] || [ -z "$fqrn" ]; then
    usage
fi

yum -y install jq sqlite

CVMFS_STRATUM0_URL=http://cvmfs-s3:9000/mybucket
CVMFS_SERVER_DEBUG=3 cvmfs_server mkfs -w $CVMFS_STRATUM0_URL/$fqrn \
                         -u gw,/srv/cvmfs/$fqrn/data/txn,$cvmfs_gateway_url:4929/api/v1 \
                         -k /etc/cvmfs/keys -o `whoami` $MKFS_OPTS $fqrn

