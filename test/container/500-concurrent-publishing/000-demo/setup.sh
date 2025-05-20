SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 1
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E $REPO_NAME || exit 2
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw1 -F $REPO_NAME -M || exit 3
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw1 -F $REPO_NAME -M || exit 4

