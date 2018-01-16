load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "datastore.go",
        "doc.go",
        "errors.go",
        "key.go",
        "load.go",
        "prop.go",
        "query.go",
        "save.go",
        "time.go",
        "transaction.go",
    ],
    importpath = "cloud.google.com/go/datastore",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/internal:go_default_library",
        "//vendor/cloud.google.com/go/internal/fields:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/timestamp:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/wrappers:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/transport/grpc:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/datastore/v1:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/type/latlng:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
        "//vendor/google.golang.org/grpc/metadata:go_default_library",
        "//vendor/google.golang.org/grpc/status:go_default_library",
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
