load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "logadmin.go",
        "metrics.go",
        "resources.go",
        "sinks.go",
    ],
    importpath = "cloud.google.com/go/logging/logadmin",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/cloud.google.com/go/logging:go_default_library",
        "//vendor/cloud.google.com/go/logging/apiv2:go_default_library",
        "//vendor/cloud.google.com/go/logging/internal:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/api/monitoredres:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/appengine/logging/v1:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/cloud/audit:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/logging/type:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/logging/v2:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
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
