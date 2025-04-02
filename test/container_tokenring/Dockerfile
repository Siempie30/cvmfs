FROM almalinux:8

# Install required dependencies
RUN dnf install -y \
    git \
    cmake \
    make \
    gcc-c++ \
    openssl-devel \
    fuse \
    fuse3-devel \
    fuse3 \
    fuse3-libs \
    fuse-devel \
    libcurl-devel \
    zlib-devel \
    pkg-config \
    python3 \
    python3-devel \
    bzip2 \
    patch \
    cpio \
    libuuid-devel \
    autofs \
    attr \
    libcap-devel \
    unzip \
    golang \
    usbutils \
    httpd
    #&& dnf clean all

# Set working directory
WORKDIR /opt

# Clone the CVMFS repository
RUN git clone https://github.com/cvmfs/cvmfs.git --depth=1

# Set build directory
WORKDIR /opt/cvmfs

# Build and install CVMFS
RUN mkdir build && cd build \
    && cmake -DBUILD_GATEWAY=ON .. \
    && make -j$(nproc) \
    && make install

# Copy the s3.conf file from the host to the container
COPY s3.conf /etc/cvmfs/s3.conf

# Start apache
RUN systemctl start httpd

# Create the repository
#RUN cvmfs_server mkfs -s /etc/cvmfs/s3.conf -w http://cvmfs-test01.s3.cern.ch test.bucket.org
RUN cvmfs_server mkfs test.repo.org

# Set default command
CMD ["/usr/bin/cvmfs_gateway"]
