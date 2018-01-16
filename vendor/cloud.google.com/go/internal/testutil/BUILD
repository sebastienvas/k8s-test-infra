load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "cmp.go",
        "context.go",
        "server.go",
        "unique.go",
    ],
    importpath = "cloud.google.com/go/internal/testutil",
    visibility = ["//vendor/cloud.google.com/go:__subpackages__"],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/google/go-cmp/cmp:go_default_library",
        "//vendor/golang.org/x/net/context:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
        "//vendor/golang.org/x/oauth2/google:go_default_library",
        "//vendor/golang.org/x/oauth2/jwt:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
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
