load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["config.go"],
    importpath = "k8s.io/test-infra/boskos/gcp",
    visibility = ["//visibility:public"],
    deps = [
        "//boskos/common:go_default_library",
        "//boskos/mason:go_default_library",
        "//vendor/github.com/sirupsen/logrus:go_default_library",
        "//vendor/golang.org/x/oauth2/google:go_default_library",
        "//vendor/golang.org/x/oauth2/jwt:go_default_library",
        "//vendor/google.golang.org/api/compute/v1:go_default_library",
        "//vendor/google.golang.org/api/container/v1:go_default_library",
        "//vendor/gopkg.in/yaml.v2:go_default_library",
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

go_test(
    name = "go_default_test",
    srcs = ["config_test.go"],
    data = ["//boskos/mason:testdata"],
    embed = [":go_default_library"],
    importpath = "k8s.io/test-infra/boskos/gcp",
    deps = ["//boskos/mason:go_default_library"],
)
