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
		return "", fmt.Errorf("error parsing old diffs %w", errOldFiles)
	}

	if errNewFiles != nil {
		return "", fmt.Errorf("error parsing new diffs %w", errNewFiles)
	}

	result := ""

	// TODO: arrays need to be sorted by filenames of origin
	// Iterate over files in FileDiff arrays
	i, j := 0, 0
	for (i < len(oldFileDiffs)) && (j < len(newFileDiffs)) {
		switch {
		case oldFileDiffs[i].OrigName == newFileDiffs[j].OrigName:
			comparedFileDiff, err := compareFileDiff(oldFileDiffs[i], newFileDiffs[j])

			if err != nil {
				return "", err
			}

			fileDiffContent, err := diff.PrintFileDiff(comparedFileDiff)
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
			result += fmt.Sprintf("Only in %s: %s\n", filepath.Dir(newFileDiffs[j].NewName),
				filepath.Base(newFileDiffs[j].NewName))
			j++
		}
	}

	// In case there are more oldFileDiffs, while newFileDiffs are run out
	for i < len(oldFileDiffs) {
		result += fmt.Sprintf("Only in %s: %s\n", filepath.Dir(oldFileDiffs[i].NewName),
			filepath.Base(oldFileDiffs[i].NewName))
		i++
	}

	// In case there are more newFileDiffs, while oldFileDiffs are run out
	for j < len(newFileDiffs) {
		result += fmt.Sprintf("Only in %s: %s\n", filepath.Dir(newFileDiffs[j].NewName),
			filepath.Base(newFileDiffs[j].NewName))
		j++
	}

	return result, nil
}

// InterDiffPath does things
func InterDiffPath(sourcePath string, oldDiff, newDiff io.Reader) (string, error) {
	return "", nil
}

func compareFileDiff(oldFileDiff, newFileDiff *diff.FileDiff) (*diff.FileDiff, error) {
	// Configuration of result FileDiff
	// TODO: something with extended (extended header lines)
	resultFileDiff := diff.FileDiff{OrigName: oldFileDiff.NewName,
		OrigTime: oldFileDiff.NewTime,
		NewName:  newFileDiff.NewName,
		NewTime:  newFileDiff.NewTime,
		Extended: []string{},
		Hunks:    []*diff.Hunk{}}

	// Iterating over hunks in order they start in origin
	i, j := 0, 0
	for (i < len(oldFileDiff.Hunks)) && (j < len(newFileDiff.Hunks)) {
		switch {
		case oldFileDiff.Hunks[i].OrigStartLine+oldFileDiff.Hunks[i].OrigLines < newFileDiff.Hunks[j].OrigStartLine:
			// Whole oldHunk is before starting of newHunk
			resultFileDiff.Hunks = append(resultFileDiff.Hunks,
				revertedHunkBody(oldFileDiff.Hunks[i]))
			i++
		case newFileDiff.Hunks[j].OrigStartLine+newFileDiff.Hunks[j].OrigLines < oldFileDiff.Hunks[i].OrigStartLine:
			// Whole newHunk is before starting of oldHunk
			resultFileDiff.Hunks = append(resultFileDiff.Hunks, newFileDiff.Hunks[j])
			j++
		default:
			// oldHunk and newHunk are overlapping somehow
			// Collecting a whole set of overlapping hunks to produce one continuous hunk
			oldHunks, newHunks := findOverlappingHunkSet(oldFileDiff, newFileDiff, &i, &j)
			mergedOverlappingHunk, err := mergeOverlappingHunks(oldHunks, newHunks)

			if err != nil {
				return nil, err
			}

			// In case opposite hunks aren't doing same changes.
			if mergedOverlappingHunk != nil {
				resultFileDiff.Hunks = append(resultFileDiff.Hunks, mergedOverlappingHunk)
			}
		}
	}

	// In case there are more hunks in oldFileDiff, while hunks of newFileDiff are run out
	for i < len(oldFileDiff.Hunks) {
		resultFileDiff.Hunks = append(resultFileDiff.Hunks,
			revertedHunkBody(oldFileDiff.Hunks[i]))
		i++
	}

	// In case there are more hunks in newFileDiff, while hunks of oldFileDiff are run out
	for j < len(newFileDiff.Hunks) {
		resultFileDiff.Hunks = append(resultFileDiff.Hunks, newFileDiff.Hunks[j])
		j++
	}

	return &resultFileDiff, nil
}

func findOverlappingHunkSet(oldFileDiff, newFileDiff *diff.FileDiff, i, j *int) (oldHunks, newHunks []*diff.Hunk) {
	// Collecting overlapped hunks into two arrays

	oldHunks = append(oldHunks, oldFileDiff.Hunks[*i])
	newHunks = append(newHunks, newFileDiff.Hunks[*j])
	*i++
	*j++

Loop:
	for {
		switch {
		// Starting line of oldHunk is in previous newHunk body (between start and last lines)
		case (*i < len(oldFileDiff.Hunks)) && (oldFileDiff.Hunks[*i].OrigStartLine >= newFileDiff.Hunks[*j-1].OrigStartLine) &&
			(oldFileDiff.Hunks[*i].OrigStartLine < newFileDiff.Hunks[*j-1].OrigStartLine+newFileDiff.Hunks[*j-1].OrigLines):
			oldHunks = append(oldHunks, oldFileDiff.Hunks[*i])
			*i++
		// Starting line of newHunk is in previous oldHunk body (between start and last lines)
		case (*j < len(newFileDiff.Hunks)) && (newFileDiff.Hunks[*j].OrigStartLine >= oldFileDiff.Hunks[*i-1].OrigStartLine) &&
			(newFileDiff.Hunks[*j].OrigStartLine < oldFileDiff.Hunks[*i-1].OrigStartLine+oldFileDiff.Hunks[*i-1].OrigLines):
			newHunks = append(newHunks, newFileDiff.Hunks[*j])
			*j++
		default:
			// No overlapping hunks left
			break Loop
		}
	}

	return oldHunks, newHunks
}

func mergeOverlappingHunks(oldHunks, newHunks []*diff.Hunk) (*diff.Hunk, error) {
	resultHunk, currentOrgI, err := configureResultHunk(oldHunks, newHunks)

	if err != nil {
		return nil, err
	}

	// Indexes of hunks
	currentOldHunkI, currentNewHunkJ := 0, 0
	// Indexes of lines in body hunks
	// if indexes == -1 -- we don't have relevant hunk, which contains changes nearby currentOrgI
	i, j := -1, -1

	// Body of hunks
	var newBody []string
	var oldHunkBody, newHunkBody []string

	// Iterating through the hunks in order there're appearing in origin file.
	// Using number of line in origin (currentOrgI) as an anchor to process line by line.
	// By using currentOrgI as anchor it is easier to see how changes have been applied step by step.

	// Merge, while there are hunks to process
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
			// Changes are only in oldHunk
			newBody = append(newBody, revertedLine(oldHunkBody[i]))
			// In case current line haven't been added, we have processed anchor line.
			if !strings.HasPrefix(oldHunkBody[i], "+") {
				// Updating index of anchor line.
				currentOrgI++
			}
			i++

		case (i == -1) && (j >= 0):
			// Changes are only in newHunk
			newBody = append(newBody, newHunkBody[j])
			// In case current line haven't been added, we have processed anchor line.
			if !strings.HasPrefix(newHunkBody[j], "+") {
				// Updating index of anchor line.
				currentOrgI++
			}
			j++

		default:
			// Changes are in old and new hunks.
			switch {
			// Firstly proceeding added lines,
			// because added lines are between previous currentOrgI and currentOrgI.
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

				// Updating currentOrgI since we have processed anchor line.
				currentOrgI++
				i++
				j++
			}
		}

		if i >= len(oldHunkBody) {
			// Proceed whole oldHunkBody
			i = -1
			currentOldHunkI++
		}

		if j >= len(newHunkBody) {
			// Proceed whole newHunkBody
			j = -1
			currentNewHunkJ++
		}
	}

	resultHunk.Body = []byte(strings.Join(newBody, "\n"))

	for _, line := range newBody {
		if !strings.HasPrefix(line, " ") {
			// resultHunkBody contains some changes
			return resultHunk, nil
		}
	}

	return nil, nil
}

func configureResultHunk(oldHunks, newHunks []*diff.Hunk) (*diff.Hunk, int32, error) {
	if (len(oldHunks) == 0) || (len(newHunks) == 0) {
		return nil, 0, fmt.Errorf("one of the hunks array is empty")
	}

	var currentOrgI int32
	resultHunk := &diff.Hunk{
		// TODO: Concatenate sections
		Section: "",
		Body:    []byte{0},
	}

	firstOldHunk, firstNewHunk := oldHunks[0], newHunks[0]
	lastOldHunk, lastNewHunk := oldHunks[len(oldHunks)-1], newHunks[len(newHunks)-1]

	// Calculate StartLine for origin and new in result
	if firstOldHunk.OrigStartLine < firstNewHunk.OrigStartLine {
		// Started with old hunk
		currentOrgI = firstOldHunk.OrigStartLine
		// As we started with this old hunk, OrigStartLine will be same as start line of hunk in old source
		resultHunk.OrigStartLine = firstOldHunk.NewStartLine
		// StartLine in firstNewHunk - number of origin lines between start of firstNewHunk and start of resultHunk
		resultHunk.NewStartLine = currentOrgI +
			firstNewHunk.NewStartLine - firstNewHunk.OrigStartLine
	} else {
		// Started with new hunk
		currentOrgI = firstNewHunk.OrigStartLine
		// StartLine in firstOldHunk - number of origin lines between start of firstOldHunk and start of resultHunk
		resultHunk.OrigStartLine = currentOrgI +
			firstOldHunk.NewStartLine - firstOldHunk.OrigStartLine
		// As we started with this new hunk, NewStartLine will be same as start line of hunk in new source
		resultHunk.NewStartLine = firstNewHunk.NewStartLine
	}

	// Calculate NumberLines for origin and new in result
	if lastOldHunk.OrigStartLine+lastOldHunk.OrigLines >
		lastNewHunk.OrigStartLine+lastNewHunk.OrigLines {
		// Finished with old hunk
		// Last line of lastOldHunk - first line of origin in resultHunk
		resultHunk.OrigLines = lastOldHunk.NewStartLine + lastOldHunk.NewLines - resultHunk.OrigStartLine
		// Last line of new in resultHunk - first line of new in resultHunk
		// lastNewHunk.NewStartLine + lastNewHunk.NewLines = last line of lastNewHunk
		resultHunk.NewLines = lastNewHunk.NewStartLine + lastNewHunk.NewLines +
			// + number of origin lines between last line of lastNewHunk and lastOldHunk
			lastOldHunk.OrigStartLine + lastOldHunk.OrigLines -
			lastNewHunk.OrigStartLine - lastNewHunk.OrigLines -
			// - first line of new in resultHunk
			resultHunk.NewStartLine
	} else {
		// Finished with new hunk
		// Last line of old in resultHunk - first line of old in resultHunk
		// lastOldHunk.NewStartLine + lastOldHunk.NewLines = last line of lastOldHunk
		resultHunk.OrigLines = lastOldHunk.NewStartLine + lastOldHunk.NewLines +
			// + number of origin lines between last line of lastOldHunk and lastNewHunk
			lastNewHunk.OrigStartLine + lastNewHunk.OrigLines -
			lastOldHunk.OrigStartLine - lastOldHunk.OrigLines -
			// - first line of old in resultHunk
			resultHunk.OrigStartLine
		// Last line of lastNewHunk - first line of new in resultHunk
		resultHunk.NewLines = lastNewHunk.NewStartLine + lastNewHunk.NewLines - resultHunk.NewStartLine
	}

	// TODO: Check those values
	resultHunk.OrigNoNewlineAt = lastOldHunk.OrigNoNewlineAt
	resultHunk.StartPosition = firstOldHunk.StartPosition

	return resultHunk, currentOrgI, nil
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
