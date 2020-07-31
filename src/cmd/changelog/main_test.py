# Copyright 2020 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import subprocess
import unittest
import json
import os


def get_filename(source, target):
    return source + ' -> ' + target + '.json'


def check_file_exists(source, target):
    return os.path.exists(get_filename(source, target))


def delete_logs(source, target):
    try:
        os.remove(get_filename(source, target))
        os.remove(get_filename(target, source))
    except OSError:
        pass


def check_empty_json_file(source, target):
    with open(get_filename(source, target)) as f:
        return f.read() == '{}'


def check_commit_schema(commit):
    schema = {
        "SHA": str,
        "AuthorName": str,
        "CommitterName": str,
        "Subject": str,
        "Bugs": list,
        "CommitTime": str,
        "ReleaseNote": str
    }
    for attr, attrType in schema.items():
        if attr not in commit:
            return False
        elif not isinstance(commit[attr], attrType):
            return False
    return True


def check_changelog_schema(source, target):
    with open(get_filename(source, target)) as f:
        data = json.load(f)
        if len(data) == 0:
            return False
        for repoName, repoLog in data.items():
            for commit in repoLog['Commits']:
                if not check_commit_schema(commit):
                    return False
    return True


class TestCLIApplication(unittest.TestCase):

    def setUp(self):
        process = subprocess.run(["go", "build", "-o", "changelog","main.go"])
        assert process.returncode == 0

    def tearDown(self):
        delete_logs("15000.0.0", "15055.0.0")
        delete_logs("15050.0.0", "15056.0.0")
        delete_logs("15056.0.0", "15056.0.0")

    def test_basic_run(self):
        source = "15050.0.0"
        target = "15056.0.0"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", source, target])
        assert process.returncode == 0
        assert check_file_exists(source, target)
        assert check_file_exists(target, source)
        assert check_changelog_schema(source, target)
        assert check_empty_json_file(target, source)

    def test_with_instance_and_repo(self):
        source = "15048.0.0"
        target = "15049.0.0"
        instance = "cos.googlesource.com"
        repo = "cos/manifest-snapshots"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", "-i", instance, "-r", repo, source, target])
        assert process.returncode == 0
        assert check_file_exists(source, target)
        assert check_file_exists(target, source)
        assert check_changelog_schema(source, target)
        assert check_empty_json_file(target, source)

    def test_large_run(self):
        source = "15055.0.0"
        target = "15000.0.0"
        instance = "cos.googlesource.com"
        repo = "cos/manifest-snapshots"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", "-i", instance, "-r", repo, source, target])
        assert process.returncode == 0
        assert check_file_exists(source, target)
        assert check_file_exists(target, source)
        assert check_changelog_schema(source, target)
        assert check_changelog_schema(target, source)

    def test_with_invalid_source(self):
        source = "99999.0.0"
        target = "15040.0.0"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", source, target])
        assert process.returncode != 0
        assert not check_file_exists(source, target)
        assert not check_file_exists(target, source)

    def test_with_invalid_target(self):
        source = "15038.0.0"
        target = "89981.0.0"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", source, target])
        assert process.returncode != 0
        assert not check_file_exists(source, target)
        assert not check_file_exists(target, source)

    def test_with_invalid_instance(self):
        source = "15048.0.0"
        target = "15049.0.0"
        instance = "cos.gglesource.com"
        repo = "cos/manifest-snapshots"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", "-i", instance, "-r", repo, source, target])
        assert process.returncode != 0
        assert not check_file_exists(source, target)
        assert not check_file_exists(target, source)

    def test_with_invalid_repo(self):
        source = "15048.0.0"
        target = "15049.0.0"
        instance = "cos.googlesource.com"
        repo = "cos/not-manifest-snapshots"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", "-i", instance, "-r", repo, source, target])
        assert process.returncode != 0
        assert not check_file_exists(source, target)
        assert not check_file_exists(target, source)

    def test_with_same_source_and_target(self):
        source = "15056.0.0"
        target = "15056.0.0"
        delete_logs(source, target)
        process = subprocess.run(["./changelog", source, target])
        assert process.returncode == 0
        assert check_file_exists(source, target)
        assert check_file_exists(target, source)
        assert check_empty_json_file(source, target)
        assert check_empty_json_file(target, source)

if __name__ == '__main__':
    unittest.main()
