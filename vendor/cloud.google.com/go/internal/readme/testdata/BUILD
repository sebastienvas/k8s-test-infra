load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "bad-no-name.go",
        "bad-no-open.go",
        "bad-unclosed.go",
        "snips.go",
    ],
    importpath = "cloud.google.com/go/internal/readme/testdata",
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
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
