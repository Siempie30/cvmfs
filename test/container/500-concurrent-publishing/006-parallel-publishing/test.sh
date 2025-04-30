UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

# Post the token to the first gateway
echo "---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

echo "\n\n---Creating subdirectories"
# Start transaction
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 2
# Create subdirectories
execute_in_container cvmfs-pub1 "mkdir -p /cvmfs/test.repo.org/1" || exit 3
execute_in_container cvmfs-pub1 "mkdir -p /cvmfs/test.repo.org/2" || exit 4
execute_in_container cvmfs-pub1 "mkdir -p /cvmfs/test.repo.org/3" || exit 5
execute_in_container cvmfs-pub1 "mkdir -p /cvmfs/test.repo.org/4" || exit 6
execute_in_container cvmfs-pub1 "mkdir -p /cvmfs/test.repo.org/5" || exit 7
# Publish changes
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 8

# Start transactions
echo "\n\n---Starting parallel transactions"
execute_in_container cvmfs-pub1 "cvmfs_server transaction test.repo.org/1" || exit 9
execute_in_container cvmfs-pub2 "cvmfs_server transaction test.repo.org/2" || exit 10
execute_in_container cvmfs-pub3 "cvmfs_server transaction test.repo.org/3" || exit 11
execute_in_container cvmfs-pub4 "cvmfs_server transaction test.repo.org/4" || exit 12
execute_in_container cvmfs-pub5 "cvmfs_server transaction test.repo.org/5" || exit 13

# Write changes in subdirectories
echo "\n\n---Writing changes in subdirectories"
execute_in_container cvmfs-pub1 "echo '1' > /cvmfs/test.repo.org/1/file1" || exit 14
execute_in_container cvmfs-pub2 "echo '2' > /cvmfs/test.repo.org/2/file2" || exit 15
execute_in_container cvmfs-pub3 "echo '3' > /cvmfs/test.repo.org/3/file3" || exit 16
execute_in_container cvmfs-pub4 "echo '4' > /cvmfs/test.repo.org/4/file4" || exit 17
execute_in_container cvmfs-pub5 "echo '5' > /cvmfs/test.repo.org/5/file5" || exit 18

sleep 3

# Publish changes
echo "\n\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 19
execute_in_container cvmfs-pub2 "cvmfs_server publish" || exit 20
execute_in_container cvmfs-pub3 "cvmfs_server publish" || exit 21
execute_in_container cvmfs-pub4 "cvmfs_server publish" || exit 22
execute_in_container cvmfs-pub5 "cvmfs_server publish" || exit 23

# Verify changes
echo "\n\n---Verifying changes"
execute_in_container cvmfs-gw1 "cvmfs_server mount" || exit 24
execute_in_container cvmfs-gw1 "cat /cvmfs/test.repo.org/1/file1 | tee | grep \"1\"" || exit 25
execute_in_container cvmfs-gw1 "cat /cvmfs/test.repo.org/2/file2 | tee | grep \"2\"" || exit 26
execute_in_container cvmfs-gw1 "cat /cvmfs/test.repo.org/3/file3 | tee | grep \"3\"" || exit 27
execute_in_container cvmfs-gw1 "cat /cvmfs/test.repo.org/4/file4 | tee | grep \"4\"" || exit 28
execute_in_container cvmfs-gw1 "cat /cvmfs/test.repo.org/5/file5 | tee | grep \"5\"" || exit 29