load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["longrunning.go"],
    importpath = "cloud.google.com/go/longrunning",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/longrunning/autogen:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/longrunning:go_default_library",
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
    srcs = [
        ":package-srcs",
        "//vendor/cloud.google.com/go/longrunning/autogen:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
