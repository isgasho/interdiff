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
		fmt.Errorf("error parsing old diffs")
		return "", errOldFiles
	}

	if errNewFiles != nil {
		fmt.Errorf("error parsing new diffs")
		return "", errNewFiles
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

			if oldFileDiff.Hunks[i].OrigStartLine < newFileDiff.Hunks[j].OrigStartLine {
				oldHunks = append(oldHunks, oldFileDiff.Hunks[i])
				i++
			} else {
				newHunks = append(newHunks, newFileDiff.Hunks[j])
				j++
			}

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
	return &resultFileDiff
}

// TODO: compare overlapped hunks
func compareOverlappedHunks(oldHunks, newHunks []*diff.Hunk) *diff.Hunk {
	if (len(oldHunks) == 0) || (len(newHunks) == 0) {
		return nil
	}

	resultHunk := &diff.Hunk{OrigStartLine: -1,
		OrigLines:       -1,
		OrigNoNewlineAt: -1,
		NewStartLine:    -1,
		NewLines:        -1,
		Section:         "",
		StartPosition:   -1,
		Body:            []byte{0},
	}

	var newBody []string

	// Indexes of hunks
	currentOldHunkI, currentNewHunkJ := 0, 0
	// Indexes of lines in body hunks
	i, j := -1, -1
	// Body of hunks
	var oldHunkBody, newHunkBody []string

	var currentOrgI int32

	// First hunk is from old ones
	if oldHunks[currentOldHunkI].OrigStartLine < newHunks[currentNewHunkJ].OrigStartLine {
		// Current number of line in origin
		currentOrgI = oldHunks[currentOldHunkI].OrigStartLine
		//oldHunkBody = strings.Split(string(oldHunks[currentOldHunkI].Body), "\n")
		//i = 0
		resultHunk.OrigStartLine = oldHunks[currentOldHunkI].NewStartLine
		resultHunk.NewStartLine = newHunks[currentNewHunkJ].NewStartLine -
			newHunks[currentNewHunkJ].OrigStartLine + currentOrgI
	} else {
		// First hunk is from new ones
		currentOrgI = newHunks[currentNewHunkJ].OrigStartLine
		//newHunkBody = strings.Split(string(newHunks[currentNewHunkJ].Body), "\n")
		//j = 0
		resultHunk.OrigStartLine = oldHunks[currentOldHunkI].NewStartLine -
			oldHunks[currentOldHunkI].OrigStartLine + currentOrgI
		resultHunk.NewStartLine = newHunks[currentNewHunkJ].NewStartLine
	}

	// Compare, while there are hunks to process
	for (currentOldHunkI < len(oldHunks)) || (currentNewHunkJ < len(newHunks)) {

		// Entering next hunk in oldHunks
		if (currentOldHunkI < len(oldHunks)) && (i == -1) && (currentOrgI == oldHunks[currentOldHunkI].OrigStartLine) {
			i = 0
			oldHunkBody = strings.Split(string(oldHunks[currentOldHunkI].Body), "\n")
		}

		// Entering next hunk in newHunks
		if (currentNewHunkJ < len(newHunks)) && (j == -1) && (currentOrgI == newHunks[currentNewHunkJ].OrigStartLine) {
			j = 0
			newHunkBody = strings.Split(string(newHunks[currentNewHunkJ].Body), "\n")
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
			if i >= len(oldHunkBody) {
				i = -1
				currentOldHunkI++
			}

		case (i == -1) && (j >= 0):
			newBody = append(newBody, newHunkBody[j])
			// Added one of lines from origin
			if !strings.HasPrefix(newHunkBody[j], "+") {
				currentOrgI++
			}
			j++
			if j >= len(newHunkBody) {
				j = -1
				currentNewHunkJ++
			}
		default:
			oldLineAction, newLineAction := oldHunkBody[i][0], newHunkBody[j][0]
			// No added lines
			if newLineAction != '+' {
				switch {
				case oldLineAction == ' ':
					switch {
					case newLineAction == ' ':
						newBody = append(newBody, oldHunkBody[i])
					case newLineAction == '-':
						newBody = append(newBody, newHunkBody[j])
					}

				case oldLineAction == '-':
					switch {
					case newLineAction == ' ':
						newBody = append(newBody, revertedLine(oldHunkBody[i]))
						// No need to proceed case since same line was deleted in both sources
						//case strings.HasPrefix(newHunkBody[j], "-"):
					}
				}
				currentOrgI++
				i++
				j++
			}

			// Contains added lines
			switch {
			case (oldLineAction == '+') && (newLineAction == '+'):
				// In case both lines have same content, then don't mark it as new
				if oldHunkBody[i][1:] == newHunkBody[j][1:] {
					newBody = append(newBody, " "+oldHunkBody[i][1:])
				} else {
					newBody = append(newBody, revertedLine(oldHunkBody[i]))
					newBody = append(newBody, newHunkBody[j])
				}
				i++
				j++
			case oldLineAction == '+':
				newBody = append(newBody, revertedLine(oldHunkBody[i]))
				i++
			case newLineAction == '+':
				newBody = append(newBody, newHunkBody[j])
				j++
			}

			if (i < len(oldHunkBody)) && (len(oldHunkBody[i]) > 0) {
				oldLineAction = oldHunkBody[i][0]
			} else {
				i = -1
				currentOldHunkI++
			}

			if (j < len(newHunkBody)) && (len(newHunkBody[j]) > 0) {
				newLineAction = newHunkBody[j][0]
			} else {
				j = -1
				currentNewHunkJ++
			}
		}
	}

	lastOldHunk, lastNewHunk := oldHunks[len(oldHunks)-1], newHunks[len(newHunks)-1]
	// Last hunk is from old ones
	if lastOldHunk.OrigStartLine+lastOldHunk.OrigLines > lastNewHunk.OrigStartLine+lastNewHunk.OrigLines {
		resultHunk.OrigLines = lastOldHunk.OrigStartLine + lastOldHunk.OrigLines - resultHunk.OrigStartLine
		resultHunk.NewLines = lastNewHunk.NewStartLine + lastNewHunk.NewLines -
			lastOldHunk.OrigStartLine - lastOldHunk.OrigLines +
			lastNewHunk.OrigStartLine + lastNewHunk.OrigLines -
			resultHunk.NewStartLine
	} else {
		// Last hunk is from new ones
		resultHunk.OrigLines = lastOldHunk.NewStartLine + lastOldHunk.NewLines -
			lastNewHunk.OrigStartLine - lastNewHunk.OrigLines +
			lastOldHunk.OrigStartLine + lastOldHunk.OrigLines -
			resultHunk.OrigStartLine
		resultHunk.NewLines = lastNewHunk.OrigStartLine + lastNewHunk.OrigLines - resultHunk.NewStartLine
	}

	resultHunk.OrigNoNewlineAt = lastOldHunk.OrigNoNewlineAt
	// TODO: Concatenate sections
	resultHunk.Section = ""
	resultHunk.StartPosition = oldHunks[0].StartPosition
	resultHunk.Body = []byte(strings.Join(newHunkBody, "\n"))

	for _, line := range newBody {
		if !strings.HasPrefix(line, " ") {
			return resultHunk
		}
	}

	return nil
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
