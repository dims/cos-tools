py_library(
    name = "release_lib",
    srcs = ["release.py"],
    data = ["release-versions.yaml"],
    srcs_version = "PY3",
)

py_binary(
    name = "release",
    srcs = ["release.py"],
    data = ["release-versions.yaml"],
    python_version = "PY3",
    deps = [":release_lib"],
)
