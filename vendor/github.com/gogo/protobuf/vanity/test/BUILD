load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["doc.go"],
    importpath = "github.com/gogo/protobuf/vanity/test",
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
    srcs = [
        ":package-srcs",
        "//vendor/github.com/gogo/protobuf/vanity/test/fast:all-srcs",
        "//vendor/github.com/gogo/protobuf/vanity/test/faster:all-srcs",
        "//vendor/github.com/gogo/protobuf/vanity/test/slick:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
