load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "admin.go",
        "bigtable.go",
        "doc.go",
        "filter.go",
        "gc.go",
        "reader.go",
    ],
    importpath = "cloud.google.com/go/bigtable",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/bigtable/internal/gax:go_default_library",
        "//vendor/cloud.google.com/go/bigtable/internal/option:go_default_library",
        "//vendor/cloud.google.com/go/longrunning:go_default_library",
        "//vendor/cloud.google.com/go/longrunning/autogen:go_default_library",
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/duration:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/api/transport/grpc:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/bigtable/admin/v2:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/bigtable/v2:go_default_library",
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
    srcs = [
        ":package-srcs",
        "//vendor/cloud.google.com/go/bigtable/bttest:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/cmd/cbt:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/cmd/emulator:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/cmd/loadtest:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/cmd/scantest:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/internal/cbtconfig:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/internal/gax:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/internal/option:all-srcs",
        "//vendor/cloud.google.com/go/bigtable/internal/stat:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
