load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["loadtest.go"],
    importpath = "cloud.google.com/go/pubsub/loadtest/cmd",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/cloud.google.com/go/pubsub/loadtest:go_default_library",
        "//vendor/cloud.google.com/go/pubsub/loadtest/pb:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
    ],
)

go_binary(
    name = "cmd",
    embed = [":go_default_library"],
    importpath = "cloud.google.com/go/pubsub/loadtest/cmd",
    visibility = ["//visibility:public"],
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
