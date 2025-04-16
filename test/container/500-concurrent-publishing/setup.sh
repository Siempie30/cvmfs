docker exec -it cvmfs-s3 /scripts/setup_s3.sh || exit $?
docker exec -it cvmfs-gw1 /scripts/setup_gateway.sh || exit $?
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh || exit $? 
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh || exit $?

