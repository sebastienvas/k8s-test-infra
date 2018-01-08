load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "fuzzy_decoder.go",
        "naming_strategy.go",
        "privat_fields.go",
        "time_as_int64_codec.go",
    ],
    importpath = "github.com/json-iterator/go/extra",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/json-iterator/go:go_default_library"],
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
