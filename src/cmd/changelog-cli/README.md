# COS Changelog

An application that generates build changelogs and locates the first build containing a CL.

## Usage

### Retrieve Changelog
Retrieve the commit changelog between two builds.

Run with `./changelog --mode changelog [options] [build-number || image-name] [build-number || image-name]`

Example: `./changelog --gob cos.googlesource.com --repo cos/manifest-snapshots cos-rc-85-13310-1034-0 15045.0.0`

### Find First Build Containing CL
Retrieve the first build containing a CL.

Run with `./changelog --mode findbuild [options] [CL-number || commit-SHA]`

Example using CL-Number: `./changelog --mode findbuild 3280`

Example using Commit-SHA: `./changelog --mode findbuild 18d4ce48c1dc2f530120f85973fec348367f78a0`

## Commands
`./changelog --help` to see a list of commands or get help for one command

## Global Options

`--mode | -m`: Specifies the query mode. Acceptable values: [changelog || findbuild]

`--gerrit URL`: (optional) Specifies the Gerrit instance to query from, with the `https://` prefix. It will use `https://cos-review.googlesource.com` by default.

`--fallback URL`: (optional) specifies the fallback Gerrit instance to query from, with the `https://` prefix. It will use `https://chromium-review.googlesource.com` by default.

`--gob URL`: (optional) Specifies the Git on Borg instance where manifest-snapshot files are located. It will use `cos.googlesource.com` by default.

`--repo`: (optional) Specifies the repository for manifest-snapshot files within the Git on Borg instance. It will use `cos/manifest-snapshots` by default.

`--debug | -d`: (optional) Enables debug messages.

## Output

## Changelog Output

Creates 2 JSON files representing the changelog between 2 given build numbers. Each output file maps repositories to their repository changelog. A repository changelog consists of all the commits in a repository that were present in between the build numbers.

All the commits that were present in the target build number and not present in the source build number are located in `source_build_number -> target_build_number.json`.

All commits that were present in the source build number but not present in the target build number are located in `target_build_num -> source_build_num.json`.

## FindCL output

Prints the first build number that includes the input CL.

## Notes
* Changelog only supports Cusky builds. For retrieving changelogs from Pre-Cusky builds, please use go/crosland.
* Changelog only supports image names satisfying the regex `^cos-(dev-|beta-|stable-|rc-)?\d+-([\d-]+)$`