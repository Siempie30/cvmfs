SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT

echo "---Starting transaction"
execute_in_container cvmfs-gw1 "cvmfs_server transaction" || exit 1
execute_in_container cvmfs-gw1 "echo 'abc' > /cvmfs/test.repo.org/gw1file" || exit 2

echo "\n---Publishing transaction"
execute_in_container cvmfs-gw1 "cvmfs_server publish" || exit 3

echo "\n---Copying repo data to host"
mkdir -p $SCRIPT_DIR/tmp
docker cp cvmfs-s3:/data/stratum0bucket/test.repo.org/data/. $SCRIPT_DIR/tmp/1 || exit 4

echo "\n---Creating gateway 2"
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw2/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E test.repo.org || exit 5

echo "\n---Verifying repository data"
docker cp cvmfs-s3:/data/stratum0bucket/test.repo.org/data/. $SCRIPT_DIR/tmp/2 || exit 6
diff --recursive $SCRIPT_DIR/tmp/1 $SCRIPT_DIR/tmp/2 || exit 7

echo "\nStarting transaction on gateway 2"
execute_in_container cvmfs-gw2 "cvmfs_server transaction" || exit 8
execute_in_container cvmfs-gw2 "echo 'def' > /cvmfs/test.repo.org/gw2file" || exit 9

echo "\nPublishing transaction on gateway 2"
execute_in_container cvmfs-gw2 "cvmfs_server publish" || exit 10

echo "\n---Copying repo data to host"
docker cp cvmfs-s3:/data/stratum0bucket/test.repo.org/data/. $SCRIPT_DIR/tmp/3 || exit 11

echo "\n---Removing repository from gateway 1"
execute_in_container cvmfs-gw1 "cvmfs_server rmfs -f test.repo.org" || exit 12

echo "\n---Verifying repository data"
docker cp cvmfs-s3:/data/stratum0bucket/test.repo.org/data/. $SCRIPT_DIR/tmp/4 || exit 13
diff --recursive $SCRIPT_DIR/tmp/3 $SCRIPT_DIR/tmp/4 || exit 14