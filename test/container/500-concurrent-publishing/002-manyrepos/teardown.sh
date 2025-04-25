docker stop cvmfs-pub1 && docker rm cvmfs-pub1 && docker volume rm 500-concurrent-publishing_var_spool_cvmfs3
docker stop cvmfs-gw1 &&  docker rm cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_cvmfs1
docker stop cvmfs-s3 && docker rm cvmfs-s3 && docker volume rm 500-concurrent-publishing_minio-data