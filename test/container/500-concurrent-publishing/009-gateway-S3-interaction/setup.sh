SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up S3
docker exec -it cvmfs-gw1 /scripts/setup_minio.sh || exit 1

# Set up gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 2

