load("@io_bazel_rules_go//go:def.bzl", "go_library")

filegroup(
    name = "go_default_library_protos",
    srcs = ["intstore.proto"],
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
)

go_library(
    name = "go_default_library",
    srcs = ["intstore.pb.go"],
    importpath = "cloud.google.com/go/internal/rpcreplay/proto/intstore",
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
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
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
