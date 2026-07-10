cmake_minimum_required(VERSION 3.20)
project(matching_engine CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_compile_options(-Wall -Wextra -Wpedantic)

# --- Core engine library (no I/O, no transport — pure matching logic) -------
add_library(engine
  src/order_book.cpp
)
target_include_directories(engine PUBLIC src)

# --- Tests -------------------------------------------------------------------
enable_testing()

# The GoogleTest suite fetches gtest over the network at configure time. Turn it
# off (-DBUILD_GTEST_TESTS=OFF) to build fully offline; the zero-dependency
# engine_check below still runs and covers the same behaviour.
option(BUILD_GTEST_TESTS "Build the GoogleTest suite (needs network to fetch gtest)" ON)

if(BUILD_GTEST_TESTS)
  include(FetchContent)
  FetchContent_Declare(
    googletest
    URL https://github.com/google/googletest/archive/refs/tags/v1.15.2.zip
    DOWNLOAD_EXTRACT_TIMESTAMP TRUE
  )
  set(INSTALL_GTEST OFF CACHE BOOL "" FORCE)
  FetchContent_MakeAvailable(googletest)

  add_executable(order_book_test tests/order_book_test.cpp)
  target_link_libraries(order_book_test PRIVATE engine GTest::gtest_main)

  include(GoogleTest)
  gtest_discover_tests(order_book_test)
endif()

# --- Zero-dependency correctness check ---------------------------------------
# Needs no test framework (no GoogleTest fetch), so it runs anywhere g++ does
# and is the fastest signal in CI. Registered with CTest alongside the gtest run.
add_executable(engine_check tests/standalone_check.cpp)
target_link_libraries(engine_check PRIVATE engine)
add_test(NAME engine_check COMMAND engine_check)

# --- Benchmark ----------------------------------------------------------------
add_executable(engine_bench bench/bench_main.cpp)
target_link_libraries(engine_bench PRIVATE engine)

# --- gRPC service wrapper -----------------------------------------------------
# The pure engine library, tests, and benchmark above have no external
# dependencies and always build (this is what CI compiles on every push).
# The gRPC server is layered on top and only builds when Protobuf + gRPC are
# discoverable, so contributors without gRPC installed still get a working
# engine + green tests. Toggle with -DBUILD_ENGINE_SERVICE=OFF.
option(BUILD_ENGINE_SERVICE "Build the gRPC matching-engine server" ON)

if(BUILD_ENGINE_SERVICE)
  find_package(Protobuf CONFIG)
  find_package(gRPC CONFIG)

  if(Protobuf_FOUND AND gRPC_FOUND)
    set(PROTO_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/../proto)
    set(PROTO_GEN_DIR ${CMAKE_CURRENT_BINARY_DIR}/generated)
    file(MAKE_DIRECTORY ${PROTO_GEN_DIR})

    # Generate C++ message + gRPC stubs from the shared contract into a known
    # directory. We invoke protoc directly instead of protobuf_generate's TARGET
    # form: the shared .proto lives outside this source tree (../proto), which
    # trips up that helper's output-path computation. Both C++ and Java still
    # generate from this same .proto — never duplicated.
    set(PROTO_GEN_SRCS
      ${PROTO_GEN_DIR}/trading.pb.cc
      ${PROTO_GEN_DIR}/trading.grpc.pb.cc)
    add_custom_command(
      OUTPUT ${PROTO_GEN_SRCS}
             ${PROTO_GEN_DIR}/trading.pb.h
             ${PROTO_GEN_DIR}/trading.grpc.pb.h
      COMMAND protobuf::protoc
      ARGS --proto_path=${PROTO_ROOT}
           --cpp_out=${PROTO_GEN_DIR}
           --grpc_out=${PROTO_GEN_DIR}
           --plugin=protoc-gen-grpc=$<TARGET_FILE:gRPC::grpc_cpp_plugin>
           ${PROTO_ROOT}/trading.proto
      DEPENDS ${PROTO_ROOT}/trading.proto
      COMMENT "Generating C++ + gRPC stubs from trading.proto"
      VERBATIM)

    add_library(trading_proto ${PROTO_GEN_SRCS})
    target_include_directories(trading_proto PUBLIC ${PROTO_GEN_DIR})
    target_link_libraries(trading_proto
      PUBLIC protobuf::libprotobuf gRPC::grpc++)

    add_executable(matching_engine_server
      src/main.cpp
      src/engine_service.cpp
      src/book_registry.cpp
      src/metrics.cpp)
    target_link_libraries(matching_engine_server
      PRIVATE engine trading_proto gRPC::grpc++)

    # --- Optional Prometheus metrics (Phase 6) --------------------------------
    # prometheus-cpp's pull exposer. If present, define HAVE_PROMETHEUS so
    # metrics.cpp serves /metrics; otherwise the engine reports to a no-op sink.
    find_package(prometheus-cpp CONFIG QUIET)
    if(prometheus-cpp_FOUND)
      message(STATUS "prometheus-cpp found — engine exposes /metrics")
      target_link_libraries(matching_engine_server
        PRIVATE prometheus-cpp::pull prometheus-cpp::core)
      target_compile_definitions(matching_engine_server PRIVATE HAVE_PROMETHEUS)
    else()
      message(STATUS "prometheus-cpp not found — engine runs without /metrics")
    endif()

    # --- Optional Kafka bridge (Phase 4) --------------------------------------
    # librdkafka's C++ API. If present, compile the bridge into the server and
    # define HAVE_KAFKA so main.cpp starts it when KAFKA_BOOTSTRAP_SERVERS is
    # set. Without it, the server is gRPC-only and still builds.
    find_path(RDKAFKACPP_INCLUDE_DIR NAMES librdkafka/rdkafkacpp.h)
    find_library(RDKAFKACPP_LIB NAMES rdkafka++)
    if(RDKAFKACPP_INCLUDE_DIR AND RDKAFKACPP_LIB)
      message(STATUS "librdkafka found — building Kafka bridge into the server")
      target_sources(matching_engine_server PRIVATE src/kafka_bridge.cpp)
      target_include_directories(matching_engine_server PRIVATE ${RDKAFKACPP_INCLUDE_DIR})
      target_link_libraries(matching_engine_server PRIVATE ${RDKAFKACPP_LIB})
      target_compile_definitions(matching_engine_server PRIVATE HAVE_KAFKA)
    else()
      message(STATUS
        "librdkafka not found — server builds gRPC-only (no Kafka bridge). "
        "Install it with 'apt install librdkafka-dev' / 'brew install librdkafka'.")
    endif()
  else()
    message(STATUS
      "gRPC/Protobuf not found — skipping matching_engine_server target. "
      "Core engine library, tests, and benchmark still build. "
      "Install gRPC (e.g. 'brew install grpc' / 'apt install libgrpc++-dev "
      "protobuf-compiler-grpc') to build the server.")
  endif()
endif()
