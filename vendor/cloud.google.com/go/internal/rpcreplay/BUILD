load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "rpcreplay.go",
    ],
    importpath = "cloud.google.com/go/internal/rpcreplay",
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
    deps = [
        "//vendor/cloud.google.com/go/internal/rpcreplay/proto/rpcreplay:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/any:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/rpc/status:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
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
        "//vendor/cloud.google.com/go/internal/rpcreplay/proto/intstore:all-srcs",
        "//vendor/cloud.google.com/go/internal/rpcreplay/proto/rpcreplay:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
