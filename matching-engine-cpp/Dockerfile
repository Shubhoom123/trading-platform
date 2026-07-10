# Multi-stage build for the C++ matching engine.
# Build context is the REPO ROOT (CMake reads ../proto):
#   docker build -f matching-engine-cpp/Dockerfile -t trading/matching-engine .
FROM ubuntu:24.04 AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential cmake git ca-certificates \
      protobuf-compiler-grpc libgrpc++-dev libprotobuf-dev librdkafka-dev \
      zlib1g-dev libcurl4-openssl-dev \
 && rm -rf /var/lib/apt/lists/*

# prometheus-cpp isn't packaged in apt; build it statically so it links into the
# engine binary (nothing extra to ship at runtime). This flips on HAVE_PROMETHEUS.
RUN git clone --depth 1 --branch v1.2.4 \
      https://github.com/jupp0r/prometheus-cpp.git /tmp/prometheus-cpp \
 && cd /tmp/prometheus-cpp && git submodule update --init \
 && cmake -B build -DBUILD_SHARED_LIBS=OFF -DENABLE_PUSH=OFF \
      -DENABLE_COMPRESSION=OFF -DENABLE_TESTING=OFF \
 && cmake --build build -j \
 && cmake --install build

WORKDIR /src
COPY proto/ proto/
COPY matching-engine-cpp/ matching-engine-cpp/

WORKDIR /src/matching-engine-cpp
RUN cmake -B build -DCMAKE_BUILD_TYPE=Release \
 && cmake --build build -j --target matching_engine_server

# Collect the binary's shared-lib dependencies (flattened into one directory)
# so the runtime image stays minimal without guessing volatile Ubuntu runtime
# package names. Flattening avoids merging onto system symlinks like /lib.
RUN mkdir -p /deps \
 && ldd /src/matching-engine-cpp/build/matching_engine_server \
      | awk 'NF==4 {print $3}' \
      | sort -u \
      | xargs -I '{}' cp -v '{}' /deps/

# Runtime: same glibc base as build, plus only the copied .so files on the
# loader path.
FROM ubuntu:24.04
COPY --from=build /deps/ /usr/local/lib/engine/
COPY --from=build /src/matching-engine-cpp/build/matching_engine_server \
     /usr/local/bin/matching_engine_server
ENV LD_LIBRARY_PATH=/usr/local/lib/engine
EXPOSE 50051 9101
ENTRYPOINT ["/usr/local/bin/matching_engine_server"]
