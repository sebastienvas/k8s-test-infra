load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "mutex.go",
        "mutex_go17.go",
        "profiler.go",
        "symbolizer.go",
    ],
    importpath = "cloud.google.com/go/profiler",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/compute/metadata:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/google/pprof/profile:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/transport/grpc:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/devtools/cloudprofiler/v2:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/rpc/errdetails:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
        "//vendor/google.golang.org/grpc/metadata:go_default_library",
        "//vendor/google.golang.org/grpc/status:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [
        ":package-srcs",
        "//vendor/cloud.google.com/go/profiler/busybench:all-srcs",
        "//vendor/cloud.google.com/go/profiler/mocks:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
