load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "fields.go",
        "fold.go",
    ],
    importpath = "cloud.google.com/go/internal/fields",
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
    deps = ["//vendor/cloud.google.com/go/internal/atomiccache:go_default_library"],
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
