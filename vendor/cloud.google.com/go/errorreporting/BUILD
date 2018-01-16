load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["errors.go"],
    importpath = "cloud.google.com/go/errorreporting",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/errorreporting/apiv1beta1:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/support/bundler:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/devtools/clouderrorreporting/v1beta1:go_default_library",
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
        "//vendor/cloud.google.com/go/errorreporting/apiv1beta1:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
