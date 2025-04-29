UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

# Post the token to the first gateway
echo "---Handing out token to gateway 1"
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
docker rm -f cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_gw1 > /dev/null 2>&1
docker rm -f cvmfs-gw2 && docker volume rm 500-concurrent-publishing_var_spool_gw2 > /dev/null 2>&1
docker-compose up -d cvmfs-test-gw1 cvmfs-test-gw2 > /dev/null 2>&1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring_gw1.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh -E $REPO_NAME > /dev/null 2>&1 || exit 4
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring_gw2.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E $REPO_NAME > /dev/null 2>&1 || exit 5
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway

# Post the token to the first gateway
echo "\n\n---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 6

sleep 3

# Check that gateway 1 has the token
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 7; fi

# Start a transaction and make changes on the publisher
echo "\n\n---Starting transaction on publisher"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 8
execute_in_container cvmfs-pub1 "echo 'abc' > /cvmfs/test.repo.org/testfile" || exit 9

sleep 3

# Check that gateway 1 has the token
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 10; fi

# Publish changes
echo "\n\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 11

# Check that gateway 1 no longer has the token, and that gateway 2 has received it.
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 12; fi
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw2:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 13; fi

# Restart the gateways
SCRIPT_DIR=$(dirname $0)
REPO_NAME=test.repo.org
echo "\n\n---Restarting gateways"
docker rm -f cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_gw1 > /dev/null 2>&1
docker rm -f cvmfs-gw2 && docker volume rm 500-concurrent-publishing_var_spool_gw2 > /dev/null 2>&1
docker-compose up -d cvmfs-test-gw1 cvmfs-test-gw2 > /dev/null 2>&1
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw1/user.json
cp $SCRIPT_DIR/token_ring_gw1.json $SCRIPT_DIR/../config/gw1/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw1 /scripts/gateway_mkfs.sh -E $REPO_NAME > /dev/null 2>&1 || exit 14
docker exec -it cvmfs-gw1 systemctl start cvmfs-gateway

cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring_gw2.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E $REPO_NAME > /dev/null 2>&1 || exit 15
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway

# Post the token to the first gateway
echo "\n\n---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 16

sleep 3

# Check that gateway 1 has the token
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 17; fi

# Start a transaction, make changes and publish on the publisher
echo "\n\n---Starting transaction on publisher"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 18
execute_in_container cvmfs-pub1 "echo 'abc' > /cvmfs/test.repo.org/testfile" || exit 19
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 20

# Check that gateway 1 has the token
has_token=$(docker exec -it cvmfs-pub1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 21; fi

sleep 2

# Check that gateway 1 no longer has the token
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 22; fi