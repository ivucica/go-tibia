# -*- mode: python; -*-
# vim: set syntax=python:

workspace(name="go_tibia")

git_repository(
    name = "rules_tibia",
    remote = "https://github.com/ivucica/rules_tibia",
    commit = "08cb15890e10856dec153a8f5e06fde5cee34d4f",
)

load("@rules_tibia//:tibia_data.bzl", "tibia_data_repository")
tibia_data_repository(version=854)

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.7.0/rules_go-0.7.0.tar.gz",
    sha256 = "91fca9cf860a1476abdc185a5f675b641b60d3acf0596679a27b580af60bf19c",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

