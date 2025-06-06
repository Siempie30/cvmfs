SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up first gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 1
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway
# Set up remaining 4 gateways
for i in $(seq 2 5); do
    cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw${i}/user.json
    cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw${i}/repo.json
    docker exec -it cvmfs-gw${i} /scripts/gateway_mkfs.sh -A $REPO_NAME || exit 2
    docker exec -it cvmfs-gw${i} systemctl start cvmfs-gateway    
done

# Set up publishers
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw1 -F $REPO_NAME -M || exit 3
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw2 -F $REPO_NAME -M || exit 4
docker exec -it cvmfs-pub3 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw3 -F $REPO_NAME -M || exit 5
docker exec -it cvmfs-pub4 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw4 -F $REPO_NAME -M || exit 6
docker exec -it cvmfs-pub5 /scripts/setup_connected_publisher.sh -G http://cvmfs-gw5 -F $REPO_NAME -M || exit 7

