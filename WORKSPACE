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

load("@io_bazel_rules_go//go:def.bzl", "go_repository")
go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    commit = "e3636079e1a4c1f337f212cc5cd2aca108f6c900",
)
go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    commit = "146acd28ed5894421fb5aac80ca93bc1b1f46f87",
)


http_file(
    name = "itemsotb854",
    url = "https://github.com/opentibia/server/raw/d5d283a6dd62a3841531428bd5e385a38d85560d/data/trunk/items/items.otb",
    sha256 = "c04ad718c90b2ea1c73234f1fd17f4ebee9df3ca9b0cdffd73f611ecb4c6937d",
)
