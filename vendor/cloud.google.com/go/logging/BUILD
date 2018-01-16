load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "logging.go",
    ],
    importpath = "cloud.google.com/go/logging",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/compute/metadata:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/cloud.google.com/go/logging/apiv2:go_default_library",
        "//vendor/cloud.google.com/go/logging/internal:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/struct:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/timestamp:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/support/bundler:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/api/monitoredres:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/logging/type:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/logging/v2:go_default_library",
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
        "//vendor/cloud.google.com/go/logging/apiv2:all-srcs",
        "//vendor/cloud.google.com/go/logging/internal:all-srcs",
        "//vendor/cloud.google.com/go/logging/logadmin:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
