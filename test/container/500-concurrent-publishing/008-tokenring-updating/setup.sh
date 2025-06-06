SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org

# Set up gateways
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh $REPO_NAME || exit 1
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw3/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw3/repo.json
docker exec -it cvmfs-gw3 /scripts/gateway_mkfs.sh -A $REPO_NAME || exit 2
docker exec -it cvmfs-gw3 systemctl start cvmfs-gateway

