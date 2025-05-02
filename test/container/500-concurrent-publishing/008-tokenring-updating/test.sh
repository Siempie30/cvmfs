SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT
TOKEN_FILE="/etc/cvmfs/gateway/token_ring.json"

echo "---Verifying initial token ring files"
EXPECTED='{
  "repoName": "test.repo.org",
  "gateways": [
    "http://cvmfs-gw1:4929/api/v1",
    "http://cvmfs-gw2:4929/api/v1"
  ]
}'
OUTPUT=$(docker exec -it cvmfs-gw1 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 1; fi
OUTPUT=$(docker exec -it cvmfs-gw2 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 2; fi

echo "\n---Creating third gateway"
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw3/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw3/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw3/repo.json
docker exec -it cvmfs-gw3 /scripts/gateway_mkfs.sh -E test.repo.org || exit 3
docker exec -it cvmfs-gw3 systemctl start cvmfs-gateway

echo "\n---Verifying updated token ring files"
EXPECTED='{
  "repoName": "test.repo.org",
  "gateways": [
    "http://cvmfs-gw1:4929/api/v1",
    "http://cvmfs-gw2:4929/api/v1",
    "http://cvmfs-gw3:4929/api/v1"
  ]
}'
OUTPUT=$(docker exec -it cvmfs-gw1 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 4; fi
OUTPUT=$(docker exec -it cvmfs-gw2 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 5; fi
OUTPUT=$(docker exec -it cvmfs-gw3 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 6; fi

echo "\n---Passing token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 7

echo "\n---Removing gateway 2"
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 8

sleep 3

echo "\n---Verifying gateway 3 has token"
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 9; fi

echo "\n---Verifying updated token ring files"
EXPECTED='{
  "repoName": "test.repo.org",
  "gateways": [
    "http://cvmfs-gw1:4929/api/v1",
    "http://cvmfs-gw3:4929/api/v1"
  ]
}'
OUTPUT=$(docker exec -it cvmfs-gw1 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 10; fi
OUTPUT=$(docker exec -it cvmfs-gw3 cat $TOKEN_FILE | jq '.repos[0]')
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 11; fi