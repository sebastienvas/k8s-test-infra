load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["doc-snippets.go"],
    importpath = "cloud.google.com/go/firestore/internal",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/firestore:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
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
