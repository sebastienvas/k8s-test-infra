load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["loadtest.go"],
    importpath = "cloud.google.com/go/pubsub/loadtest",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/pubsub:go_default_library",
        "//vendor/cloud.google.com/go/pubsub/loadtest/pb:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/golang.org/x/time/rate:go_default_library",
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
        "//vendor/cloud.google.com/go/pubsub/loadtest/cmd:all-srcs",
        "//vendor/cloud.google.com/go/pubsub/loadtest/pb:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
