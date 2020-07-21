package interdiff

import (
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"io"
	"path/filepath"
	"strings"
)

// RollupDiff does things
func RollupDiff(oldSource, newSource, diff io.Reader) (string, error) {
	return "", nil
}

// RollupDiffPath does things
func RollupDiffPath(oldSourcePath, newSourcePath string, diff io.Reader) (string, error) {
	return "", nil
}

// InterDiff looks for difference between two diff files.
func InterDiff(oldDiff, newDiff io.Reader) (string, error) {
	oldFileDiffs, errOldFiles := diff.NewMultiFileDiffReader(oldDiff).ReadAllFiles()
	newFileDiffs, errNewFiles := diff.NewMultiFileDiffReader(newDiff).ReadAllFiles()

	if errOldFiles != nil {
		return "", fmt.Errorf("error parsing old diffs %s", errOldFiles)
	}

	if errNewFiles != nil {
		return "", fmt.Errorf("error parsing new diffs %s", errNewFiles)
	}

	result := ""

	// TODO: arrays need to be sorted by filenames of origin
	i, j := 0, 0
	for (i < len(oldFileDiffs)) && (j < len(newFileDiffs)) {
		switch {
		case oldFileDiffs[i].OrigName == newFileDiffs[j].OrigName:
			fileDiffContent, err := diff.PrintFileDiff(compareFileDiff(oldFileDiffs[i], newFileDiffs[j]))
			if err == nil {
				result += string(fileDiffContent)
			} else {
				return "", err
			}
			i++
			j++
		case oldFileDiffs[i].OrigName < newFileDiffs[j].OrigName:
			result += fmt.Sprintf("Only in %s: %s\n", filepath.Dir(oldFileDiffs[i].NewName),
				filepath.Base(oldFileDiffs[i].NewName))
			i++
		default:
			result += fmt.Sprintf("Only in %s: %s\n", filepath.Dir(newFileDiffs[i].NewName),
				filepath.Base(newFileDiffs[j].NewName))
			j++
		}
	}

	return result, nil
}

// InterDiffPath does things
func InterDiffPath(sourcePath string, oldDiff, newDiff io.Reader) (string, error) {
	return "", nil
}

func compareFileDiff(oldFileDiff, newFileDiff *diff.FileDiff) *diff.FileDiff {
	// TODO: something with extended (extended header lines)
	resultFileDiff := diff.FileDiff{OrigName: oldFileDiff.NewName,
		OrigTime: oldFileDiff.NewTime,
		NewName:  newFileDiff.NewName,
		NewTime:  newFileDiff.NewTime,
		Extended: []string{},
		Hunks:    []*diff.Hunk{}}

	i, j := 0, 0
	for (i < len(oldFileDiff.Hunks)) && (j < len(newFileDiff.Hunks)) {
		switch {
		case oldFileDiff.Hunks[i].OrigStartLine+oldFileDiff.Hunks[i].OrigLines < newFileDiff.Hunks[j].OrigStartLine:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks,
				revertedHunkBody(oldFileDiff.Hunks[i]))
			i++
		case newFileDiff.Hunks[j].OrigStartLine+newFileDiff.Hunks[j].OrigLines < oldFileDiff.Hunks[i].OrigStartLine:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks, newFileDiff.Hunks[j])
			j++
		default:
			// Collecting overlapped hunks into two arrays
			var oldHunks, newHunks []*diff.Hunk

			oldHunks = append(oldHunks, oldFileDiff.Hunks[i])
			newHunks = append(newHunks, newFileDiff.Hunks[j])
			i++
			j++

			findAll := false

			for !findAll {
				switch {
				// Starting line of old hunk is in new hunk body
				case (i < len(oldFileDiff.Hunks)) && (oldFileDiff.Hunks[i].OrigStartLine >= newFileDiff.Hunks[j-1].OrigStartLine) &&
					(oldFileDiff.Hunks[i].OrigStartLine < newFileDiff.Hunks[j-1].OrigStartLine+newFileDiff.Hunks[j-1].OrigLines):
					oldHunks = append(oldHunks, oldFileDiff.Hunks[i])
					i++
				// Starting line of new hunk is in old hunk body
				case (j < len(newFileDiff.Hunks)) && (newFileDiff.Hunks[j].OrigStartLine >= oldFileDiff.Hunks[i-1].OrigStartLine) &&
					(newFileDiff.Hunks[j].OrigStartLine < oldFileDiff.Hunks[i-1].OrigStartLine+oldFileDiff.Hunks[i-1].OrigLines):
					newHunks = append(newHunks, newFileDiff.Hunks[j])
					j++
				default:
					findAll = true
				}
			}

			comparedFileDiff := compareOverlappedHunks(oldHunks, newHunks)
			// Body of hunks aren't same.
			if comparedFileDiff != nil {
				resultFileDiff.Hunks = append(resultFileDiff.Hunks, comparedFileDiff)
			}
		}
	}

	for i < len(oldFileDiff.Hunks) {
		resultFileDiff.Hunks = append(resultFileDiff.Hunks,
			revertedHunkBody(oldFileDiff.Hunks[i]))
		i++
	}

	for j < len(newFileDiff.Hunks) {
		resultFileDiff.Hunks = append(resultFileDiff.Hunks, newFileDiff.Hunks[j])
		j++
	}

	return &resultFileDiff
}

func compareOverlappedHunks(oldHunks, newHunks []*diff.Hunk) *diff.Hunk {
	if (len(oldHunks) == 0) || (len(newHunks) == 0) {
		return nil
	}

	resultHunk, currentOrgI := configureResultHunk(oldHunks, newHunks)

	if resultHunk == nil {
		return nil
	}

	// Indexes of hunks
	currentOldHunkI, currentNewHunkJ := 0, 0
	// Indexes of lines in body hunks
	i, j := -1, -1

	// Body of hunks
	var newBody []string
	var oldHunkBody, newHunkBody []string

	// Compare, while there are hunks to process
	for (currentOldHunkI < len(oldHunks)) || (currentNewHunkJ < len(newHunks)) {

		// Entering next hunk in oldHunks
		if (currentOldHunkI < len(oldHunks)) && (i == -1) && (currentOrgI == oldHunks[currentOldHunkI].OrigStartLine) {
			i = 0
			oldHunkBody = strings.Split(string(oldHunks[currentOldHunkI].Body), "\n")
			// Remove empty line in the end
			if oldHunkBody[len(oldHunkBody)-1] == "" {
				oldHunkBody = oldHunkBody[:len(oldHunkBody)-1]
			}
		}

		// Entering next hunk in newHunks
		if (currentNewHunkJ < len(newHunks)) && (j == -1) && (currentOrgI == newHunks[currentNewHunkJ].OrigStartLine) {
			j = 0
			newHunkBody = strings.Split(string(newHunks[currentNewHunkJ].Body), "\n")
			if newHunkBody[len(newHunkBody)-1] == "" {
				newHunkBody = newHunkBody[:len(newHunkBody)-1]
			}
		}

		switch {
		case (i == -1) && (j == -1):
			break
		case (i >= 0) && (j == -1):
			newBody = append(newBody, revertedLine(oldHunkBody[i]))
			// Added one of lines from origin
			if !strings.HasPrefix(oldHunkBody[i], "+") {
				currentOrgI++
			}
			i++

		case (i == -1) && (j >= 0):
			newBody = append(newBody, newHunkBody[j])
			// Added one of lines from origin
			if !strings.HasPrefix(newHunkBody[j], "+") {
				currentOrgI++
			}
			j++

		default:
			switch {
			// Firstly proceeding added lines
			case strings.HasPrefix(oldHunkBody[i], "+"):
				newBody = append(newBody, revertedLine(oldHunkBody[i]))
				i++
			case strings.HasPrefix(newHunkBody[j], "+"):
				newBody = append(newBody, newHunkBody[j])
				j++
			default:
				switch {
				case strings.HasPrefix(oldHunkBody[i], " ") && strings.HasPrefix(newHunkBody[j], " "):
					newBody = append(newBody, oldHunkBody[i])
				case strings.HasPrefix(oldHunkBody[i], "-") && strings.HasPrefix(newHunkBody[j], " "):
					newBody = append(newBody, revertedLine(oldHunkBody[i]))
				case strings.HasPrefix(oldHunkBody[i], " ") && strings.HasPrefix(newHunkBody[j], "-"):
					newBody = append(newBody, newHunkBody[j])
					// If both have deleted same line, no need to append it to newBody
				}

				currentOrgI++
				i++
				j++
			}
		}

		if i >= len(oldHunkBody) {
			i = -1
			currentOldHunkI++
		}

		if j >= len(newHunkBody) {
			j = -1
			currentNewHunkJ++
		}
	}

	resultHunk.Body = []byte(strings.Join(newBody, "\n"))

	for _, line := range newBody {
		if !strings.HasPrefix(line, " ") {
			return resultHunk
		}
	}

	return nil
}

func configureResultHunk(oldHunks, newHunks []*diff.Hunk) (*diff.Hunk, int32) {
	if (len(oldHunks) == 0) || (len(newHunks) == 0) {
		return nil, 0
	}

	var currentOrgI int32
	resultHunk := &diff.Hunk{OrigStartLine: -1,
		OrigLines:       -1,
		OrigNoNewlineAt: -1,
		NewStartLine:    -1,
		NewLines:        -1,
		// TODO: Concatenate sections
		Section:       "",
		StartPosition: -1,
		Body:          []byte{0},
	}

	firstOldHunk, firstNewHunk := oldHunks[0], newHunks[0]
	lastOldHunk, lastNewHunk := oldHunks[len(oldHunks)-1], newHunks[len(newHunks)-1]

	// Calculate StartLine for origin and new in result
	// Started with old hunk
	if firstOldHunk.OrigStartLine < firstNewHunk.OrigStartLine {
		currentOrgI = firstOldHunk.OrigStartLine
		resultHunk.OrigStartLine = firstOldHunk.NewStartLine
		resultHunk.NewStartLine = firstNewHunk.NewStartLine -
			firstNewHunk.OrigStartLine + currentOrgI
	} else {
		// Started with new hunk
		currentOrgI = firstNewHunk.OrigStartLine
		resultHunk.OrigStartLine = firstOldHunk.NewStartLine -
			firstOldHunk.OrigStartLine + currentOrgI
		resultHunk.NewStartLine = firstNewHunk.NewStartLine
	}

	// Calculate NumberLines for origin and new in result
	// Finished with old hunk
	if lastOldHunk.OrigStartLine+lastOldHunk.OrigLines > lastNewHunk.OrigStartLine+lastNewHunk.OrigLines {
		resultHunk.OrigLines = lastOldHunk.NewStartLine + lastOldHunk.NewLines - resultHunk.OrigStartLine
		resultHunk.NewLines = lastNewHunk.NewStartLine + lastNewHunk.NewLines +
			lastOldHunk.OrigStartLine + lastOldHunk.OrigLines -
			lastNewHunk.OrigStartLine - lastNewHunk.OrigLines -
			resultHunk.NewStartLine
	} else {
		// Finished with new hunk
		resultHunk.OrigLines = lastOldHunk.NewStartLine + lastOldHunk.NewLines +
			lastNewHunk.OrigStartLine + lastNewHunk.OrigLines -
			lastOldHunk.OrigStartLine - lastOldHunk.OrigLines -
			resultHunk.OrigStartLine
		resultHunk.NewLines = lastNewHunk.NewStartLine + lastNewHunk.NewLines - resultHunk.NewStartLine
	}

	// TODO: Check those values
	resultHunk.OrigNoNewlineAt = lastOldHunk.OrigNoNewlineAt
	resultHunk.StartPosition = firstOldHunk.StartPosition

	return resultHunk, currentOrgI
}

func revertedHunkBody(hunk *diff.Hunk) *diff.Hunk {
	var newBody []string

	lines := strings.Split(string(hunk.Body), "\n")

	for _, line := range lines {
		newBody = append(newBody, revertedLine(line))
	}

	revertedHunk := &diff.Hunk{OrigStartLine: hunk.OrigStartLine,
		OrigLines:       hunk.OrigLines,
		OrigNoNewlineAt: hunk.OrigNoNewlineAt,
		NewStartLine:    hunk.NewStartLine,
		NewLines:        hunk.NewLines,
		Section:         hunk.Section,
		StartPosition:   hunk.StartPosition,
		Body:            []byte(strings.Join(newBody, "\n")),
	}

	return revertedHunk
}

func revertedLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+"):
		return "-" + line[1:]
	case strings.HasPrefix(line, "-"):
		return "+" + line[1:]
	default:
		return line
	}
}
