load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "goimports.go",
        "language.go",
        "linewriter.go",
        "main.go",
        "render_client.go",
        "render_constants.go",
        "render_provider.go",
        "render_server.go",
        "render_types.go",
        "renderer.go",
    ],
    importpath = "github.com/googleapis/gnostic/plugins/gnostic-go-generator",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/github.com/googleapis/gnostic/plugins:go_default_library",
        "//vendor/github.com/googleapis/gnostic/surface:go_default_library",
    ],
)

go_binary(
    name = "gnostic-go-generator",
    embed = [":go_default_library"],
    importpath = "github.com/googleapis/gnostic/plugins/gnostic-go-generator",
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
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/googleauth:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/v2.0/apis_guru:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/v2.0/bookstore/service:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/v2.0/xkcd:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/v3.0/bookstore/service:all-srcs",
        "//vendor/github.com/googleapis/gnostic/plugins/gnostic-go-generator/examples/v3.0/urlshortener:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
