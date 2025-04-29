SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up first gateway
docker exec -it cvmfs-gw1 /scripts/setup_minio.sh || exit 1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 2
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway
# Set up remaining 4 gateways
for i in $(seq 2 5); do
    cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw${i}/user.json
    cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw${i}/token_ring.json
    cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw${i}/repo.json
    docker exec -it cvmfs-gw${i} /scripts/gateway_mkfs.sh -E $REPO_NAME || exit 3
    docker exec -it cvmfs-gw${i} systemctl start cvmfs-gateway    
done

# Set up publishers
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 4
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh http://cvmfs-gw2 $REPO_NAME || exit 5
docker exec -it cvmfs-pub3 /scripts/setup_connected_publisher.sh http://cvmfs-gw3 $REPO_NAME || exit 6
docker exec -it cvmfs-pub4 /scripts/setup_connected_publisher.sh http://cvmfs-gw4 $REPO_NAME || exit 7
docker exec -it cvmfs-pub5 /scripts/setup_connected_publisher.sh http://cvmfs-gw5 $REPO_NAME || exit 8

