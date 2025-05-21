SCRIPT_DIR=$(dirname $0)

# Set up gateway
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway || exit 1
