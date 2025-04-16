docker exec -it cvmfs-gw1 /scripts/setup_gateway.sh || exit $?
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh || exit $? 
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh || exit $?

