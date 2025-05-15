SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT

echo "\n---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

echo "\n---Verifying gateway 1 has token"
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 2; fi

echo "\n---Handing out invalid token to gateway 1"
response=$(docker exec -it cvmfs-gw1 curl -s -X POST --data '{"repo":"fake.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.acknowledgement')
if [ "$response" != "\"error\"" ]; then exit 3; fi

sleep 2

echo "\n---Handing out token to gateway 1 (again)"
response=$(docker exec -it cvmfs-gw1 curl -s -X POST --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.acknowledgement')
if [ "$response" != "\"error\"" ]; then exit 4; fi

sleep 1

echo "\n---Verifying gateway 1 no longer has token"
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 5; fi

