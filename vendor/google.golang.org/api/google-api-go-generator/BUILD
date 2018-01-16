load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "docurls.go",
        "gen.go",
    ],
    importpath = "google.golang.org/api/google-api-go-generator",
    visibility = ["//visibility:private"],
    deps = ["//vendor/google.golang.org/api/google-api-go-generator/internal/disco:go_default_library"],
)

go_binary(
    name = "google-api-go-generator",
    embed = [":go_default_library"],
    importpath = "google.golang.org/api/google-api-go-generator",
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
        "//vendor/google.golang.org/api/google-api-go-generator/internal/disco:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
