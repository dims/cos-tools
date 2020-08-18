package binary

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// findOSConfigs finds the list of directory entries in the /etc of both images
// and stores them into the binaryDiff struct
/// Input:
//   (*ImageInfo) image1 - Holds needed rootfs path for image 1
//   (*ImageInfo) image2 - Holds needed rootfs path
// Output:
//   (*Differences) binaryDiff - Holds needed KernelCommandLine map
func findOSConfigs(image1, image2 *input.ImageInfo, binaryDiff *Differences) error {
	etcFiles1, err := ioutil.ReadDir(image1.RootfsPartition3 + etc)
	if err != nil {
		return fmt.Errorf("fail to read contents of directory %v: %v", image1.RootfsPartition3+etc, err)
	}
	etcEntries1 := []string{}
	for _, f := range etcFiles1 {
		etcEntries1 = append(etcEntries1, f.Name())
	}

	etcFiles2, err := ioutil.ReadDir(image2.RootfsPartition3 + etc)
	if err != nil {
		return fmt.Errorf("fail to read contents of directory %v: %v", image2.RootfsPartition3+etc, err)
	}
	etcEntries2 := []string{}
	for _, f := range etcFiles2 {
		etcEntries2 = append(etcEntries2, f.Name())
	}

	osConfigsMap := make(map[string]string)
	for _, elem1 := range etcEntries1 {
		if !utilities.InArray(elem1, etcEntries2) { // Unique file or directory in image 1
			osConfigsMap[elem1] = image1.TempDir
		} else { // Common /etc files or directories for image 1 and 2
			osConfigsMap[elem1] = ""
		}
	}
	for _, elem2 := range etcEntries2 {
		if _, ok := osConfigsMap[elem2]; !ok { // Unique file or directory in image 2
			osConfigsMap[elem2] = image2.TempDir
		}
	}
	binaryDiff.OSConfigs = osConfigsMap
	return nil
}

// findDiffDir finds the directory name from the "diff" command
// for the "Only in [file path]" case.
// Input:
//   (string) line - A single line of output from the "diff -rq" command
//   (string) dir1 - Path to directory 1
//   (string) dir2 - Path to directory 2
//   (bool) ok - Flag to indicate a directory has been found
// Output:
//   (string) dir1 or dir2 - The directory found in "line"
func findDiffDir(line, dir1, dir2 string) (string, bool) {
	lineSplit := strings.Split(line, " ")
	if len(lineSplit) < 3 {
		return "", false
	}

	for _, word := range lineSplit {
		if strings.Contains(word, dir1) && strings.Contains(word, dir2) {
			return "", false
		}
		if strings.Contains(word, dir1) {
			return dir1, true
		}
		if strings.Contains(word, dir2) {
			return dir2, true
		}
	}
	return "", false
}

// compressString compresses lines of a string that fit a pattern
// Input:
//   (string) dir1 - Path to directory 1
//   (string) dir2 - Path to directory 2
//   (string) root - Name of the root for directories 1 and 2
//   (string) input - The string to be filtered
//   ([]string) patterns - The list of patterns to be filtered out
// Output:
//   (string) output - The compacted version of the input string
func compressString(dir1, dir2, root, input string, patterns []string) (string, error) {
	patternMap := utilities.SliceToMapStr(patterns)

	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		for pat, count := range patternMap {
			fullPattern := filepath.Join(root, pat)
			fileInPattern := fullPattern + "/"
			onlyInPattern := fullPattern + ":"
			if strings.Contains(line, fileInPattern) || strings.Contains(line, onlyInPattern) {
				lineSplit := strings.Split(line, " ")
				if len(lineSplit) < 3 {
					continue
				}

				typeOfDiff := lineSplit[0]
				if typeOfDiff == "Files" || typeOfDiff == "Symbolic" || typeOfDiff == "File" {
					if strings.Contains(count, "differentFilesFound") {
						lines[i] = ""
						continue
					}
					lines[i] = "Files in " + filepath.Join(dir1, pat) + " and " + filepath.Join(dir2, pat) + " differ"
					patternMap[pat] += "differentFilesFound"
				} else if typeOfDiff == "Only" {
					if strings.Contains(count, "dir1_UniqueFileFound") && strings.Contains(count, "dir2_UniqueFileFound") {
						lines[i] = ""
						continue
					}
					if onlyDir, ok := findDiffDir(line, dir1, dir2); ok {
						if onlyDir == dir1 {
							if !strings.Contains(count, "dir1_UniqueFileFound") {
								lines[i] = "Unique files in " + filepath.Join(onlyDir, pat)
								patternMap[pat] += "dir1_UniqueFileFound"
							} else {
								lines[i] = ""
								continue
							}
						} else if onlyDir == dir2 {
							if !strings.Contains(count, "dir2_UniqueFileFound") {
								lines[i] = "Unique files in " + filepath.Join(onlyDir, pat)
								patternMap[pat] += "dir2_UniqueFileFound"
							} else {
								lines[i] = ""
								continue
							}
						}
					}
				} else { // Compress any other diff output not described above
					lines[i] = ""
				}
			}
		}
	}

	output := strings.Join(lines, "\n")
	output = regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(output), "\n")
	return output, nil
}

// DirectoryDiff finds the recursive file difference between two directories.
// If verbose is true return full difference, else compress based on compressedDirs
// Input:
//   (string) dir1 - Path to directory 1
//   (string) dir2 - Path to directory 2
//   (string) root - Name of the root for directories 1 and 2
//   ([]string) compressedDirs - List of directories to compress by
//   (bool) verbose - Flag that determines whether to show full or compressed difference
// Output:
//   (string) diff - The file difference output of the "diff" command
func directoryDiff(dir1, dir2, root string, verbose bool, compressedDirs []string) (string, error) {
	diff, err := exec.Command("sudo", "diff", "--no-dereference", "-rq", "-x", "etc", dir1, dir2).Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 2 {
			return "", fmt.Errorf("failed to call 'diff' command on directories %v and %v: %v", dir1, dir2, err)
		}
	}
	diffStrln := fmt.Sprintf("%s", diff)
	diffStr := strings.TrimSuffix(diffStrln, "\n")
	if verbose {
		return diffStr, nil
	}
	compressedDiffStr, err := compressString(dir1, dir2, root, diffStr, compressedDirs)
	if err != nil {
		return "", fmt.Errorf("failed to call compress 'diff' output between %v and %v: %v", dir1, dir2, err)
	}
	return compressedDiffStr, nil
}

// pureDiff returns the output of a normal diff between two files or directories
func pureDiff(input1, input2 string) (string, error) {
	diff, err := exec.Command("sudo", "diff", input1, input2).Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 2 {
			return "", fmt.Errorf("failed to call 'diff' on %v and %v: %v", input1, input2, err)
		}
	}
	diffStrln := fmt.Sprintf("%s", diff)
	diffStr := strings.TrimSuffix(diffStrln, "\n")
	return diffStr, nil
}
