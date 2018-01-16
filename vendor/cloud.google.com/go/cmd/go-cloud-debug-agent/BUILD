load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = select({
        "@io_bazel_rules_go//go/platform:linux": [
            "debuglet.go",
        ],
        "//conditions:default": [],
    }),
    importpath = "cloud.google.com/go/cmd/go-cloud-debug-agent",
    visibility = ["//visibility:private"],
    deps = select({
        "@io_bazel_rules_go//go/platform:linux": [
            "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/breakpoints:go_default_library",
            "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/controller:go_default_library",
            "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/valuecollector:go_default_library",
            "//vendor/cloud.google.com/go/compute/metadata:go_default_library",
            "//vendor/golang.org/x/debug:go_default_library",
            "//vendor/golang.org/x/debug/local:go_default_library",
            "//vendor/golang.org/x/net/context:go_default_library",
            "//vendor/golang.org/x/oauth2:go_default_library",
            "//vendor/golang.org/x/oauth2/google:go_default_library",
            "//vendor/google.golang.org/api/clouddebugger/v2:go_default_library",
        ],
        "//conditions:default": [],
    }),
)

go_binary(
    name = "go-cloud-debug-agent",
    embed = [":go_default_library"],
    importpath = "cloud.google.com/go/cmd/go-cloud-debug-agent",
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
        "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/breakpoints:all-srcs",
        "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/controller:all-srcs",
        "//vendor/cloud.google.com/go/cmd/go-cloud-debug-agent/internal/valuecollector:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
