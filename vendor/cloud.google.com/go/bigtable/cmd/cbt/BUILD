load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "cbt.go",
        "cbtdoc.go",
    ],
    importpath = "cloud.google.com/go/bigtable/cmd/cbt",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/cloud.google.com/go/bigtable:go_default_library",
        "//vendor/cloud.google.com/go/bigtable/internal/cbtconfig:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
    ],
)

go_binary(
    name = "cbt",
    embed = [":go_default_library"],
    importpath = "cloud.google.com/go/bigtable/cmd/cbt",
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
