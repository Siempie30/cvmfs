SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 1
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

# Set up publisher
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw1 -F $REPO_NAME -M || exit 2

