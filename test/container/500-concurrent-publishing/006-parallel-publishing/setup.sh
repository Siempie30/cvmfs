SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up S3
bash $SCRIPT_DIR/../scripts/setup_minio.sh || exit 1

# Set up gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 2
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

# Set up publishers
docker exec -it cvmfs-pub1 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 3
docker exec -it cvmfs-pub1 sh -c "echo CVMFS_MULTIPLE_GATEWAYS=true >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"
docker exec -it cvmfs-pub1 cvmfs_config reload || exit 4
docker exec -it cvmfs-pub2 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 5
docker exec -it cvmfs-pub2 sh -c "echo CVMFS_MULTIPLE_GATEWAYS=true >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"
docker exec -it cvmfs-pub2 cvmfs_config reload || exit 6
docker exec -it cvmfs-pub3 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 7
docker exec -it cvmfs-pub3 sh -c "echo CVMFS_MULTIPLE_GATEWAYS=true >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"
docker exec -it cvmfs-pub3 cvmfs_config reload || exit 8
docker exec -it cvmfs-pub4 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 9
docker exec -it cvmfs-pub4 sh -c "echo CVMFS_MULTIPLE_GATEWAYS=true >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"
docker exec -it cvmfs-pub4 cvmfs_config reload || exit 10
docker exec -it cvmfs-pub5 /scripts/setup_connected_publisher.sh http://cvmfs-gw1 $REPO_NAME || exit 11
docker exec -it cvmfs-pub5 sh -c "echo CVMFS_MULTIPLE_GATEWAYS=true >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"
docker exec -it cvmfs-pub5 cvmfs_config reload || exit 12

