UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

# Post the token to the first gateway
echo "\n\n---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

sleep 3

# Check that gateway 1 has the token
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 2; fi

sleep 3

# Check that gateway 1 no longer has the token
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 3; fi

# Restart the gateways
SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org
echo "\n\n---Restarting gateways"
docker rm -f cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_gw1
docker rm -f cvmfs-gw2 && docker volume rm 500-concurrent-publishing_var_spool_gw2
docker-compose up -d cvmfs-test-gw1 cvmfs-test-gw2
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring_gw1.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh -E $REPO_NAME || exit 2
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring_gw2.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E $REPO_NAME || exit 3
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway

