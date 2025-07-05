load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_andybons_gogif",
        importpath = "github.com/andybons/gogif",
        sum = "h1:WBBv0ka2SO7Ut4bpskb87E9cHNnJabqA6VoBTex0Jng=",
        version = "v0.0.0-20140526152223-16d573594812",
    )
    go_repository(
        name = "com_github_bourgeoisbear_rasterm",
        importpath = "github.com/BourgeoisBear/rasterm",
        sum = "h1:k3/mcjyo3ukAkMA2PDdtrBGv16NvJ26ABd9p9hIzbp8=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_bradfitz_iter",
        importpath = "github.com/bradfitz/iter",
        sum = "h1:GKTyiRCL6zVf5wWaqKnf+7Qs6GbEPfd4iMOitWzXJx8=",
        version = "v0.0.0-20191230175014-e8f45d346db8",
    )
    go_repository(
        name = "com_github_common_nighthawk_go_figure",
        importpath = "github.com/common-nighthawk/go-figure",
        sum = "h1:J5BL2kskAlV9ckgEsNQXscjIaLiOYiZ75d4e94E6dcQ=",
        version = "v0.0.0-20210622060536-734e95fb86be",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:ZDRjVQ15GmhC3fiQ8ni8+OwkZQO4DARzQgrnXU1Liz8=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_ericpauley_go_quantize",
        importpath = "github.com/ericpauley/go-quantize",
        sum = "h1:BBade+JlV/f7JstZ4pitd4tHhpN+w+6I+LyOS7B4fyU=",
        version = "v0.0.0-20200331213906-ae555eb2afa4",
    )
    go_repository(
        name = "com_github_felixge_httpsnoop",
        importpath = "github.com/felixge/httpsnoop",
        sum = "h1:lvB5Jl89CsZtGIWuTcDM1E/vkVs49/Ml7JJe07l8SPQ=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_golang_glog",
        importpath = "github.com/golang/glog",
        sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
        version = "v0.0.0-20160126235308-23def4e6c14b",
    )
    go_repository(
        name = "com_github_golang_jwt_jwt",
        importpath = "github.com/golang-jwt/jwt",
        sum = "h1:IfV12K8xAKAnZqdXVzCZ+TOjboZ2keLg81eXfW3O+oY=",
        version = "v3.2.2+incompatible",
    )
    go_repository(
        name = "com_github_gookit_color",
        importpath = "github.com/gookit/color",
        sum = "h1:2Si0/JAEE2+1hkNYuTszu54Ti9wfp+M4JNNrknf9/D0=",
        version = "v1.2.3",
    )
    go_repository(
        name = "com_github_gorilla_handlers",
        importpath = "github.com/gorilla/handlers",
        sum = "h1:9lRY6j8DEeeBT10CvO9hGW0gmky0BprnvDI5vfhUHH4=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_gorilla_mux",
        importpath = "github.com/gorilla/mux",
        sum = "h1:zoNxOV7WjqXptQOVngLmcSQgXmgk4NMz1HibBchjl/I=",
        version = "v1.7.2",
    )
    go_repository(
        name = "com_github_mattn_gowasmer",
        importpath = "github.com/mattn/gowasmer",
        sum = "h1:DSRlbxik5+xC6hzQ0Imr1Bef+rP1TspFoZUvWHNYG3c=",
        version = "v0.0.0-20220518070401-e6bdba3bec84",
    )
    go_repository(
        name = "com_github_nfnt_resize",
        importpath = "github.com/nfnt/resize",
        sum = "h1:zYyBkD/k9seD2A7fsi6Oo2LfFZAehjjQMERAvZLEDnQ=",
        version = "v0.0.0-20180221191011-83c6a9932646",
    )
    go_repository(
        name = "com_github_pkg_errors",
        importpath = "github.com/pkg/errors",
        sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
        version = "v0.9.1",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_sherclockholmes_webpush_go",
        importpath = "github.com/SherClockHolmes/webpush-go",
        sum = "h1:sGv0/ZWCvb1HUH+izLqrb2i68HuqD/0Y+AmGQfyqKJA=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:4G4v2dO3VZwixGIRoQ5Lfboy6nUhCyYzaqnIAPPhYs4=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:nwc3DEeHmmLAfoZucVR881uASk0Mfjw8xYJ99tb5CcY=",
        version = "v1.7.0",
    )
    go_repository(
        name = "com_github_vincent_petithory_dataurl",
        importpath = "github.com/vincent-petithory/dataurl",
        sum = "h1:cXw+kPto8NLuJtlMsI152irrVw9fRDX8AbShPRpg2CI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_wasmerio_wasmer_go",
        importpath = "github.com/wasmerio/wasmer-go",
        sum = "h1:MnqHoOGfiQ8MMq2RF6wyCeebKOe84G88h5yv+vmxJgs=",
        version = "v1.0.4",
    )
    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=",
        version = "v0.0.0-20161208181325-20d25e280405",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:dUUwHk2QECo/6vqA44rthZ8ie2QXMNeKRTHCNY2nXvo=",
        version = "v3.0.0-20200313102051-9f266ea9e77c",
    )
    go_repository(
        name = "net_badc0de_pkg_flagutil",
        importpath = "badc0de.net/pkg/flagutil",
        sum = "h1:0ZgBzd3FehDUA8DJ70/phsnDH61/3aYMyx8Wd84KqQo=",
        version = "v1.0.1",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:Roh6XWxHFKrPgC/EQhVubSAGQ6Ozk6IdxHSzt1mR0EI=",
        version = "v0.0.0-20220112180741-5e0467b6c7ce",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:CIJ76btIcR3eFI5EgSo6k1qKw9KJexJuRLI9G7Hp5wE=",
        version = "v0.0.0-20211112202133-69e39bad7dc2",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:qwRHBd0NqMbJxfbotnDhm2ByMI1Shq4Y6oRJo21SGJA=",
        version = "v0.0.0-20200625203802-6e8e738ad208",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:SrN+KX8Art/Sf4HNj6Zcz06G7VEz+7w9tdXTPOZ7+l4=",
        version = "v0.0.0-20210615035016-665e8c7367d1",
    )
    go_repository(
        name = "org_golang_x_term",
        importpath = "golang.org/x/term",
        sum = "h1:SZxvLBoTP5yHO3Frd4z4vrF+DBX9vMVanchswa69toE=",
        version = "v0.0.0-20210220032956-6a3ed077a48d",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:aRYxNxv6iGQlyVaZmk6ZgYEDa+Jg18DxebPSrd6bg1M=",
        version = "v0.3.6",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:FDhOuMEY4JVRztM/gsbk+IKUQ8kj74bxZrgw87eMMVc=",
        version = "v0.0.0-20180917221912-90fa682c2a6e",
    )
