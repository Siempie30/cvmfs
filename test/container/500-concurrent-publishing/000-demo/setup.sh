SCRIPT_DIR=$(dirname $0)
docker exec -it cvmfs-gw1 /scripts/setup_minio.sh || exit 1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring_gw1.json $SCRIPT_DIR/../config/gw1/token_ring.json
docker exec -it cvmfs-gw1 /scripts/setup_gateway.sh || exit 2
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring_gw2.json $SCRIPT_DIR/../config/gw2/token_ring.json
docker exec -it cvmfs-gw2 /scripts/setup_gateway.sh -E || exit 3
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 || exit 4
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 || exit 5

