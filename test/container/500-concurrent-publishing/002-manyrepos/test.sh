UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

echo "\n\n---Creating 100 repos"
for i in $(seq 100); do
  execute_in_container cvmfs-gw1 "/scripts/gateway_mkfs.sh ${i}test.repo.org" || exit 1
done

echo "\n\n---Adding repos to publisher"
for i in $(seq 100); do
  execute_in_container cvmfs-pub1 "/scripts/setup_connected_publisher.sh -G http://cvmfs-gw1 -F ${i}test.repo.org" || exit 2
done

echo "\n\n---Starting transactions"
for i in $(seq 100); do
  execute_in_container cvmfs-pub1 "cvmfs_server transaction ${i}test.repo.org" || exit 4
done

echo "\n\n---Writing changes"
for i in $(seq 100); do
  execute_in_container cvmfs-pub1 "echo '${i}' > /cvmfs/${i}test.repo.org/testFile" || exit 5
done

echo "\n\n---Publishing changes"
for i in $(seq 100); do
  execute_in_container cvmfs-pub1 "cvmfs_server publish ${i}test.repo.org" || exit 6
done

echo "\n\n---Checking content"
for i in $(seq 100); do
  execute_in_container cvmfs-gw1 "cvmfs_server mount ${i}test.repo.org; cat /cvmfs/${i}test.repo.org/testFile" || exit 7
  execute_in_container cvmfs-gw1 "cat /cvmfs/${i}test.repo.org/testFile | tee | grep \"${i}\"" || exit 8
done

exit 0
