SCRIPT_DIR=$(dirname $0)

# Set up S3
bash $SCRIPT_DIR/../scripts/setup_minio.sh || exit 1

# Set up gateways
# Gateway 1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring_gw1.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo_gw1.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh test1.repo.org || exit 2
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh test2.repo.org || exit 3
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway
# Gateway 2
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring_gw2.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo_gw2.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E test1.repo.org || exit 4
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh test3.repo.org || exit 5
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway
# Gateway 3
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw3/user.json
cp $SCRIPT_DIR/token_ring_gw3.json $SCRIPT_DIR/../config/gw3/token_ring.json
cp $SCRIPT_DIR/repo_gw3.json $SCRIPT_DIR/../config/gw3/repo.json
docker exec -it cvmfs-gw3 /scripts/gateway_mkfs.sh -E test2.repo.org || exit 6
docker exec -it cvmfs-gw3 /scripts/gateway_mkfs.sh -E test3.repo.org || exit 7
docker exec -it cvmfs-gw3 systemctl start cvmfs-gateway


# Set up publishers
# Publisher 1
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 test1.repo.org || exit 8
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw2 test3.repo.org || exit 9
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw3 test2.repo.org || exit 10
# Publisher 2
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 test2.repo.org || exit 11
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh http://cvmfs-gw2 test1.repo.org || exit 12
# Publisher 3
docker exec -it cvmfs-pub3 /scripts/setup_connected_publisher.sh http://cvmfs-gw2 test1.repo.org || exit 13
docker exec -it cvmfs-pub3 /scripts/setup_connected_publisher.sh http://cvmfs-gw3 test3.repo.org || exit 14

