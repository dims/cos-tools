# COS Changelog

An application that generates build changelogs and locates the first build containing a CL.

## Usage

### Retrieve Changelog
Run with `./changelog --mode changelog [options] source_build_number target_build_number`

Example: `./changelog --gob cos.googlesource.com --repo cos/manifest-snapshots 15037.0.0 15045.0.0`

### Find First Build Containing CL
Run with `./changelog --mode findbuild [options] CL-number or commit SHA`

Example: `./changelog --mode findbuild 2781`

## Commands
`./changelog --help` to see a list of commands or get help for one command

## Global Options

`--mode | -m`: Specifies the query mode. Acceptable values: changelog | findbuild

`--gerrit URL`: (optional) Specifies the Gerrit instance to query from, with the `https://` prefix. It will use `https://cos-review.googlesource.com` by default.

`--fallback URL`: (optional) specifies the fallback Gerrit instance to query from, with the `https://` prefix. It will use `https://chromium-review.googlesource.com` by default.

`--gob URL`: (optional) Specifies the Git on Borg instance where manifest-snapshot files are located. It will use `cos.googlesource.com` by default.

`--repo`: (optional) Specifies the repository for manifest-snapshot files within the Git on Borg instance. It will use `cos/manifest-snapshots` by default.

`--prefix`: (optional) Specifies a repository prefix for CLs retrieved from the fallback Gerrit instance. For example, a prefix of `mirrors/cros/` for a CL in repository `chromiumos/overlays/chromiumos-overlay` will instruct the program to search the repository `mirrors/cros/chromiumos/overlays/chromiumos-overlay` in Git on Borg. It will use `mirrors/cros/` by default.

`--debug | -d`: (optional) Enables debug messages.

## Output

## Changelog Output

Creates 2 JSON files representing the changelog between 2 given build numbers. Each output file maps repositories to their repository changelog. A repository changelog consists of all the commits in a repository that were present in between the build numbers.

All the commits that were present in the target build number and not present in the source build number are located in `source_build_number -> target_build_number.json`.

All commits that were present in the source build number but not present in the target build number are located in `target_build_num -> source_build_num.json`.

## FindCL output

Prints the first build number that includes the input CL.

## Notes
* Changelog only supports querying from build numbers from [COS GoB](cos.googlesource.com). It does not support build numbers generated using [ChromiumOS GoB](https://chromium.googlesource.com/).