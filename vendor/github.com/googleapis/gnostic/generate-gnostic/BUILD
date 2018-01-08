load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "domain.go",
        "generate-compiler.go",
        "generate-extension.go",
        "generate-proto.go",
        "helpers.go",
        "main.go",
        "types.go",
    ],
    importpath = "github.com/googleapis/gnostic/generate-gnostic",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/github.com/googleapis/gnostic/compiler:go_default_library",
        "//vendor/github.com/googleapis/gnostic/jsonschema:go_default_library",
        "//vendor/github.com/googleapis/gnostic/printer:go_default_library",
    ],
)

go_binary(
    name = "generate-gnostic",
    embed = [":go_default_library"],
    importpath = "github.com/googleapis/gnostic/generate-gnostic",
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
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
