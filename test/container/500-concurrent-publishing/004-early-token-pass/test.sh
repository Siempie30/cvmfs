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
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 4
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 5
execute_in_container cvmfs-gw1 "systemctl start cvmfs-gateway" || exit 6
execute_in_container cvmfs-gw2 "systemctl start cvmfs-gateway" || exit 7

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

sleep 6

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
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 14
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 15
execute_in_container cvmfs-gw1 "systemctl start cvmfs-gateway" || exit 16
execute_in_container cvmfs-gw2 "systemctl start cvmfs-gateway" || exit 17

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