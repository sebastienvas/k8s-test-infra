load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["translate.go"],
    importpath = "cloud.google.com/go/translate",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/cloud.google.com/go/translate/internal/translate/v2:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/golang.org/x/text/language:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/transport/http:go_default_library",
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
        "//vendor/cloud.google.com/go/translate/internal/translate/v2:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
