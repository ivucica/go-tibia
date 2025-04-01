# -*- mode: python; -*-
# vim: set syntax=python:

workspace(name="go_tibia")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

local_repository(
    name = "rules_tibia",
    path = __workspace_dir__ + "/nonvendor/github.com/ivucica/rules_tibia",
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
go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    commit = "91708ff8e35bafc8612f690a25f5dd0be6f16864",
)
go_repository(
    name = "com_github_gorilla_handlers",
    importpath = "github.com/gorilla/handlers",
    commit = "3e030244b4ba0480763356fc8ca0ade6222e2da0",
)
go_repository(
    name = "net_badc0de_pkg_flagutil",
    importpath = "badc0de.net/pkg/flagutil",
    commit = "36598d429b78588385e87050ffdd3b5a646d0c62",
)
go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    commit = "0976fa681c295de5355f7a4d968b56cb9da8a76b",
)
go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    commit = "5dd12d0cfe7f152f80558d591504ce685299311e",
)
go_repository(
    name = "com_github_ericpauley_go_quantize",
    importpath = "github.com/ericpauley/go-quantize",
    commit = "ae555eb2afa4d069c3a75cff344528ed1e9acf85",
)
go_repository(
    name = "com_github_SherClockHolmes_webpush_go",
    importpath = "github.com/SherClockHolmes/webpush-go",
    commit = "9f057bcddd4d4ee494899af787cd67cfa868c80b",
)
go_repository(
    name = "com_github_golang_jwt_jwt",
    importpath = "github.com/golang-jwt/jwt",
    commit = "dbeaa9332f19a944acb5736b4456cfcc02140e29",
)
go_repository(
    name = "com_github_BourgeoisBear_rasterm",
    importpath = "github.com/BourgeoisBear/rasterm",
    commit = "0ce274fef5b22acc1a9c84e7667250f5d4d7f22e",
)
go_repository(
    name = "org_golang_x_term",
    importpath = "golang.org/x/term",
    commit = "e5f449aeb1717c7a4156811d5d65510fee9d1530",
)
go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    commit = "bc2c85ada10aa9b6aa9607e9ac9ad0761b95cf1d",
)
go_repository(
    name = "com_github_bradfitz_iter",
    importpath = "github.com/bradfitz/iter",
    commit = "e8f45d346db8021e0dd53899bf55eb6e21218b33",
)
go_repository(
    name = "com_github_andybons_gogif",
    importpath = "github.com/andybons/gogif",
    commit = "16d573594812bc09bc62ad1d8a4129c7ba885dc6",
)
go_repository(
    name = "com_github_gookit_color",
    importpath = "github.com/gookit/color",
    #commit = "d5924d1101229c714b390f2acd9a812aa02b1e7f", # 2022
    commit = "a0d845dddbc22f444bf1fa7068c3b8bf52cd0f06", # 2018 (avoiding math.Round from go 1.10)
)
go_repository(
    name = "com_github_nfnt_resize",
    importpath = "github.com/nfnt/resize",
    commit = "83c6a9932646f83e3267f353373d47347b6036b2",
)
go_repository(
    name = "com_github_felixge_httpsnoop",
    importpath = "github.com/felixge/httpsnoop",
    commit = "ef9fc62cdc3cc5abc33d6018fe1324890bb48145",
)
go_repository(
    name = "com_github_xo_terminfo",
    importpath = "github.com/xo/terminfo",
    commit = "ca9a967f877831dd8742c136f5c19f82d03673f4",
)

http_file(
    name = "itemsotb854",
    url = "https://github.com/opentibia/server/raw/d5d283a6dd62a3841531428bd5e385a38d85560d/data/trunk/items/items.otb",
    sha256 = "c04ad718c90b2ea1c73234f1fd17f4ebee9df3ca9b0cdffd73f611ecb4c6937d",
)
