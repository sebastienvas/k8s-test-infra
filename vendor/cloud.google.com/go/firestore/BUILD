load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "collref.go",
        "doc.go",
        "docref.go",
        "document.go",
        "fieldpath.go",
        "from_value.go",
        "options.go",
        "query.go",
        "to_value.go",
        "transaction.go",
        "writebatch.go",
    ],
    importpath = "cloud.google.com/go/firestore",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/firestore/apiv1beta1:go_default_library",
        "//vendor/cloud.google.com/go/internal/fields:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes:go_default_library",
        "//vendor/github.com/golang/protobuf/ptypes/wrappers:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/firestore/v1beta1:go_default_library",
        "//vendor/google.golang.org/genproto/googleapis/type/latlng:go_default_library",
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
        "//vendor/cloud.google.com/go/firestore/apiv1beta1:all-srcs",
        "//vendor/cloud.google.com/go/firestore/genproto:all-srcs",
        "//vendor/cloud.google.com/go/firestore/internal:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
