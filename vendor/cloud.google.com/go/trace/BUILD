load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "grpc.go",
        "http.go",
        "sampling.go",
        "trace.go",
    ],
    importpath = "cloud.google.com/go/trace",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/internal/tracecontext:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/golang.org/x/time/rate:go_default_library",
        "//vendor/google.golang.org/api/cloudtrace/v1:go_default_library",
        "//vendor/google.golang.org/api/gensupport:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/support/bundler:go_default_library",
        "//vendor/google.golang.org/api/transport/http:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/metadata:go_default_library",
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
        "//vendor/cloud.google.com/go/trace/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/trace/apiv2:all-srcs",
        "//vendor/cloud.google.com/go/trace/testdata/helloworld:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
