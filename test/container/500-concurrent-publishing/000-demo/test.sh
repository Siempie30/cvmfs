SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT
REPO_NAME=test.repo.org

echo "\n---Transaction 1: pub 1, unimportant change"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 2
execute_in_container cvmfs-pub1 "echo abc > /cvmfs/$REPO_NAME/testfile"

echo "\n---Publishing transaction 1"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 3

echo "\n---Transaction 2: pub 1, important change"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 4
execute_in_container cvmfs-pub1 "echo important_change > /cvmfs/$REPO_NAME/testfile" || exit 5

echo "\n\n---Transaction 3: pub 2, unimportant change"
execute_in_container cvmfs-pub2 "cvmfs_server transaction -t 5000" &
sleep 1

echo "\n\n---Publishing transaction 2"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 7
wait $(jobs -p) 
sleep 1

execute_in_container cvmfs-pub2 "echo unrelated_change > /cvmfs/$REPO_NAME/another_file" || exit 8

echo "\n\n---Publishing transaction 3"
execute_in_container cvmfs-pub2 "cvmfs_server publish" || exit 9

docker exec cvmfs-gw1 cvmfs_server tag -l
docker exec cvmfs-gw1 cat /cvmfs/$REPO_NAME/another_file
docker exec cvmfs-gw1 cat /cvmfs/$REPO_NAME/testfile
docker exec cvmfs-gw1 cat /cvmfs/$REPO_NAME/testfile | tee | grep important_change ||  echo -e ERROR >&2

echo "\n\n---Stopping the gateway"
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 10

echo "\n\n---Setting multigateway mode on gateway and publisher"
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
jq '.repos |= map(. + {token_ring: ["cvmfs-gw1:4929/api/v1", "cvmfs-gw2:4929/api/v1"]})' $SCRIPT_DIR/repo.json > $SCRIPT_DIR/../config/gw1/repo.json
jq '.repos |= map(. + {token_ring: ["cvmfs-gw1:4929/api/v1", "cvmfs-gw2:4929/api/v1"]})' $SCRIPT_DIR/repo.json > $SCRIPT_DIR/../config/gw2/repo.json
jq '. + {"enable_multi_gateway": "true"}' $SCRIPT_DIR/user.json > $SCRIPT_DIR/../config/gw1/user.json
jq '. + {"enable_multi_gateway": "true"}' $SCRIPT_DIR/user.json > $SCRIPT_DIR/../config/gw2/user.json
execute_in_container cvmfs-pub1 "echo \"CVMFS_MULTIPLE_GATEWAYS=true\" >> /etc/cvmfs/repositories.d/$REPO_NAME/server.conf" || exit 11

echo "\n\n---Starting second gateway"
execute_in_container cvmfs-gw2 "/scripts/gateway_mkfs.sh -E $REPO_NAME" || exit 12
execute_in_container cvmfs-gw2 "systemctl start cvmfs-gateway" || exit 13

execute_in_container cvmfs-gw1 "systemctl start cvmfs-gateway" || exit 14

echo "\n\n---Passing token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"$REPO_NAME\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 15

echo "\n\n---Starting transaction on publisher 1"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 16
execute_in_container cvmfs-pub1 "echo def > /cvmfs/$REPO_NAME/testfile" || exit 17

echo "\n\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 18

echo "\n\n---Verifying changes on gateway 2"
execute_in_container cvmfs-gw2 "cvmfs_server mount; cat /cvmfs/$REPO_NAME/testfile" || exit 19
docker exec  cvmfs-gw2 cat /cvmfs/$REPO_NAME/testfile | tee | grep def ||  exit 20

echo "\n\n---Stopping both gateways"
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 21
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 22

echo "\n\n---Disabling multi-gateway mode on gateway 1 and publisher"
jq 'del(.enable_multi_gateway)' $SCRIPT_DIR/../config/gw1/user.json > tmp.json && mv tmp.json $SCRIPT_DIR/../config/gw1/user.json
jq '.repos |= map(del(.token_ring))' $SCRIPT_DIR/../config/gw1/repo.json > tmp.json && mv tmp.json $SCRIPT_DIR/../config/gw1/repo.json
execute_in_container cvmfs-pub1 "sed -i '/^CVMFS_MULTIPLE_GATEWAYS=true$/d' /etc/cvmfs/repositories.d/$REPO_NAME/server.conf"

echo "\n\n---Starting gateway 1"
execute_in_container cvmfs-gw1 "systemctl start cvmfs-gateway" || exit 23

echo "\n\n---Starting transaction on publisher 1"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 24
execute_in_container cvmfs-pub1 "echo ghi > /cvmfs/$REPO_NAME/testfile" || exit 25

echo "\n\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 26

echo "\n\n---Verifying changes on gateway 1"
execute_in_container cvmfs-gw1 "cvmfs_server mount; cat /cvmfs/$REPO_NAME/testfile" || exit 27
docker exec  cvmfs-gw1 cat /cvmfs/$REPO_NAME/testfile | tee | grep ghi ||  exit 28
