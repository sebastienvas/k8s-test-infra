load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["busybench.go"],
    importpath = "cloud.google.com/go/profiler/busybench",
    visibility = ["//visibility:private"],
    deps = ["//vendor/cloud.google.com/go/profiler:go_default_library"],
)

go_binary(
    name = "busybench",
    embed = [":go_default_library"],
    importpath = "cloud.google.com/go/profiler/busybench",
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
