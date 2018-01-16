load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["cloud.go"],
    importpath = "cloud.google.com/go",
    visibility = ["//visibility:public"],
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
        "//vendor/cloud.google.com/go/bigquery:all-srcs",
        "//vendor/cloud.google.com/go/bigtable:all-srcs",
        "//vendor/cloud.google.com/go/civil:all-srcs",
        "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent:all-srcs",
        "//vendor/cloud.google.com/go/compute/metadata:all-srcs",
        "//vendor/cloud.google.com/go/container:all-srcs",
        "//vendor/cloud.google.com/go/datastore:all-srcs",
        "//vendor/cloud.google.com/go/debugger/apiv2:all-srcs",
        "//vendor/cloud.google.com/go/dlp/apiv2beta1:all-srcs",
        "//vendor/cloud.google.com/go/errorreporting:all-srcs",
        "//vendor/cloud.google.com/go/firestore:all-srcs",
        "//vendor/cloud.google.com/go/iam:all-srcs",
        "//vendor/cloud.google.com/go/internal:all-srcs",
        "//vendor/cloud.google.com/go/language/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/language/apiv1beta2:all-srcs",
        "//vendor/cloud.google.com/go/logging:all-srcs",
        "//vendor/cloud.google.com/go/longrunning:all-srcs",
        "//vendor/cloud.google.com/go/monitoring/apiv3:all-srcs",
        "//vendor/cloud.google.com/go/profiler:all-srcs",
        "//vendor/cloud.google.com/go/pubsub:all-srcs",
        "//vendor/cloud.google.com/go/spanner:all-srcs",
        "//vendor/cloud.google.com/go/speech/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/speech/apiv1beta1:all-srcs",
        "//vendor/cloud.google.com/go/storage:all-srcs",
        "//vendor/cloud.google.com/go/trace:all-srcs",
        "//vendor/cloud.google.com/go/translate:all-srcs",
        "//vendor/cloud.google.com/go/videointelligence/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/videointelligence/apiv1beta1:all-srcs",
        "//vendor/cloud.google.com/go/videointelligence/apiv1beta2:all-srcs",
        "//vendor/cloud.google.com/go/vision/apiv1:all-srcs",
        "//vendor/cloud.google.com/go/vision/apiv1p1beta1:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
