load("@io_bazel_rules_go//go:def.bzl", "go_library")

filegroup(
    name = "go_default_library_protos",
    srcs = ["map.proto"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["map.pb.go"],
    importpath = "github.com/gogo/protobuf/test/mapdefaults",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/gogo/protobuf/proto:go_default_library"],
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
        "//vendor/github.com/gogo/protobuf/test/mapdefaults/combos/both:all-srcs",
        "//vendor/github.com/gogo/protobuf/test/mapdefaults/combos/marshaler:all-srcs",
        "//vendor/github.com/gogo/protobuf/test/mapdefaults/combos/neither:all-srcs",
        "//vendor/github.com/gogo/protobuf/test/mapdefaults/combos/unmarshaler:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
