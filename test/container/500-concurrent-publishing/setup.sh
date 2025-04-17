docker exec -it cvmfs-gw1 /scripts/setup_gateway.sh || exit 1
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh || exit 2
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh || exit 3

