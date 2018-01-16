load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery.go",
        "copy.go",
        "dataset.go",
        "doc.go",
        "error.go",
        "external.go",
        "extract.go",
        "file.go",
        "gcs.go",
        "iterator.go",
        "job.go",
        "load.go",
        "params.go",
        "query.go",
        "schema.go",
        "table.go",
        "uploader.go",
        "value.go",
    ],
    importpath = "cloud.google.com/go/bigquery",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/cloud.google.com/go/civil:go_default_library",
        "//vendor/cloud.google.com/go/internal:go_default_library",
        "//vendor/cloud.google.com/go/internal/atomiccache:go_default_library",
        "//vendor/cloud.google.com/go/internal/fields:go_default_library",
        "//vendor/cloud.google.com/go/internal/optional:go_default_library",
        "//vendor/cloud.google.com/go/internal/version:go_default_library",
        "//vendor/github.com/googleapis/gax-go:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/google.golang.org/api/bigquery/v2:go_default_library",
        "//vendor/google.golang.org/api/googleapi:go_default_library",
        "//vendor/google.golang.org/api/iterator:go_default_library",
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
        "//vendor/cloud.google.com/go/bigquery/datatransfer/apiv1:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
