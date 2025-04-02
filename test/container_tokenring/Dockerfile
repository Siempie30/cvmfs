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
    golang
    #&& dnf clean all

# Set working directory
WORKDIR /opt

# Clone the CVMFS repository
RUN git clone https://github.com/cvmfs/cvmfs.git --depth=1

# Set build directory
WORKDIR /opt/cvmfs

# Create a build directory and switch to it
RUN mkdir build && cd build \
    && cmake -DBUILD_GATEWAY=ON .. \
    && make -j$(nproc)

# Set default command
CMD ["/bin/bash"]
