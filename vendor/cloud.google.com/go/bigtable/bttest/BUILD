load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["inmem.go"],
    importpath = "cloud.google.com/go/bigtable/bttest",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/golang/protobuf/ptypes/empty:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/wrappers:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/bigtable/admin/v2:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/bigtable/v2:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/rpc/status:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
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
