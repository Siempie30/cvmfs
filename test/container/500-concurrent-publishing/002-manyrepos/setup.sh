SCRIPT_DIR=$(dirname $0)
bash $SCRIPT_DIR/../scripts/setup_minio.sh || exit 1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway || exit 2
