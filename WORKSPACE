# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the License);
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an AS IS BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "355d40d12749d843cfd05e14c304ac053ae82be4cd257efaf5ef8ce2caf31f1c",
    strip_prefix = "rules_go-197699822e081dad064835a09825448a3e4cc2a2",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/archive/197699822e081dad064835a09825448a3e4cc2a2.tar.gz",
        "https://github.com/bazelbuild/rules_go/archive/197699822e081dad064835a09825448a3e4cc2a2.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
    ],
)

http_archive(
    name = "rules_pkg",
    sha256 = "aeca78988341a2ee1ba097641056d168320ecc51372ef7ff8e64b139516a4937",
    urls = ["https://github.com/bazelbuild/rules_pkg/releases/download/0.2.6-1/rules_pkg-0.2.6.tar.gz"],
)

git_repository(
    name = "com_google_protobuf",
    commit = "31ebe2ac71400344a5db91ffc13c4ddfb7589f92",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1591135967 -0700",
)

git_repository(
    name = "com_github_googlecloudplatform_docker_credential_gcr",
    commit = "6093d30b51d725877bc6971aa6700153c1a364f1",
    remote = "https://github.com/GoogleCloudPlatform/docker-credential-gcr",
    shallow_since = "1613169008 -0800",
    patches = [
        "//src/third_party/docker_credential_gcr:0001-Add-explicit-targets-for-amd64-and-arm64.patch",
    ],
    patch_args = ["-p1"],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains(version="1.16")

load("//:deps.bzl", "go_mod_deps")

# gazelle:repository_macro deps.bzl%go_mod_deps
go_mod_deps()

rules_pkg_dependencies()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
