# COS Changelog

An application that generates a changelog between 2 build numbers based on the commit difference between them.

## Usage

Run with `./changelog [global_options] source_build_number target_build_number`

Example: `./changelog --instance cos.googlesource.com --repo cos/manifest-snapshots 15037.0.0 15045.0.0`

## Commands
`./changelog --help` to see a list of commands or get help for one command

## Global Options

`--instance | -i`: (optional) Specifies the Git on Borg instance where manifest-snapshot files are located. It will use "cos.googlesource.com" by default if not specified.

`--repo | -r`: (optional) Specifies the repository for manifest-snapshot files within the Git on Borg instance. It will use "cos/manifest-snapshots" if not specified.

## Output

Creates 2 JSON files representing the changelog between 2 given build numbers. Each output file maps repositories to their repository changelog. A repository changelog consists of all the commits in a repository that were present in between the build numbers.

All the commits that were present in the target build number and not present in the source build number are located in `source_build_number -> target_build_number.json`.

All commits that were present in the source build number but not present in the target build number are located in `target_build_num -> source_build_num.json`.