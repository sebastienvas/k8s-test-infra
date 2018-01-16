load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "backoff.go",
        "client.go",
        "doc.go",
        "errors.go",
        "go18.go",
        "key.go",
        "mutation.go",
        "not_go18.go",
        "protoutils.go",
        "read.go",
        "retry.go",
        "row.go",
        "session.go",
        "statement.go",
        "timestampbound.go",
        "transaction.go",
        "util.go",
        "value.go",
    ],
    importpath = "cloud.google.com/go/spanner",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/civil:go_default_library",
        "//vendor/cloud.google.com/go/internal/fields:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/duration:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/struct:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/timestamp:go_default_library",
        "//vendor/go.opencensus.io/plugin/grpc:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/transport/grpc:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/rpc/errdetails:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/spanner/v1:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
        "//vendor/google.golang.org/grpc/codes:go_default_library",
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
        "//vendor/cloud.google.com/go/spanner/admin/database/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/spanner/admin/instance/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/spanner/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/spanner/internal/testutil:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
