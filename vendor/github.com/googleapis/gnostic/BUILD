load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["gnostic.go"],
    importpath = "github.com/googleapis/gnostic",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/googleapis/gnostic/OpenAPIv2:go_default_library",
        "//vendor/github.com/googleapis/gnostic/OpenAPIv3:go_default_library",
        "//vendor/github.com/googleapis/gnostic/compiler:go_default_library",
        "//vendor/github.com/googleapis/gnostic/discovery:go_default_library",
        "//vendor/github.com/googleapis/gnostic/jsonwriter:go_default_library",
        "//vendor/github.com/googleapis/gnostic/plugins:go_default_library",
        "//vendor/github.com/googleapis/gnostic/surface:go_default_library",
        "//vendor/gopkg.in/yaml.v2:go_default_library",
    ],
)

go_binary(
    name = "gnostic",
    embed = [":go_default_library"],
    importpath = "github.com/googleapis/gnostic",
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
        "//vendor/github.com/googleapis/gnostic/OpenAPIv2:all-srcs",
        "//vendor/github.com/googleapis/gnostic/OpenAPIv3:all-srcs",
        "//vendor/github.com/googleapis/gnostic/apps/disco:all-srcs",
        "//vendor/github.com/googleapis/gnostic/apps/petstore-builder:all-srcs",
        "//vendor/github.com/googleapis/gnostic/apps/report:all-srcs",
        "//vendor/github.com/googleapis/gnostic/compiler:all-srcs",
        "//vendor/github.com/googleapis/gnostic/discovery:all-srcs",
        "//vendor/github.com/googleapis/gnostic/extensions:all-srcs",
        "//vendor/github.com/googleapis/gnostic/generate-gnostic:all-srcs",
        "//vendor/github.com/googleapis/gnostic/jsonschema:all-srcs",
        "//vendor/github.com/googleapis/gnostic/jsonwriter:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins:all-srcs",
        "//vendor/github.com/googleapis/gnostic/printer:all-srcs",
        "//vendor/github.com/googleapis/gnostic/surface:all-srcs",
        "//vendor/github.com/googleapis/gnostic/tools/format-schema:all-srcs",
        "//vendor/github.com/googleapis/gnostic/tools/j2y2j:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
