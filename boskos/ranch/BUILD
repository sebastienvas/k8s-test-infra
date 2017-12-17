package(default_visibility = ["//visibility:public"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
    "go_test",
)

go_test(
    name = "go_default_test",
    srcs = ["ranch_test.go"],
    embed = [":go_default_library"],
    importpath = "k8s.io/test-infra/boskos/ranch",
    deps = [
        "//boskos/common:go_default_library",
        "//boskos/crds:go_default_library",
    ],
)

go_library(
    name = "go_default_library",
    srcs = [
        "crd_storage.go",
        "memory_storage.go",
        "ranch.go",
    ],
    importpath = "k8s.io/test-infra/boskos/ranch",
    deps = [
        "//boskos/common:go_default_library",
        "//boskos/crds:go_default_library",
        "//vendor/github.com/deckarep/golang-set:go_default_library",
        "//vendor/github.com/hashicorp/go-multierror:go_default_library",
        "//vendor/github.com/sirupsen/logrus:go_default_library",
        "//vendor/gopkg.in/yaml.v2:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
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
)
