package interdiff

import (
	"bytes"
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
	// TODO: something with extendent (extended header lines)
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
		case newFileDiff.Hunks[i].OrigStartLine+newFileDiff.Hunks[i].OrigLines < oldFileDiff.Hunks[j].OrigStartLine:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks, newFileDiff.Hunks[j])
			j++
		// Hunks are overlapping
		default:
			comparedFileDiff := compareHunk(oldFileDiff.Hunks[i], newFileDiff.Hunks[j])
			// Body of hunks aren't same.
			if comparedFileDiff != nil {
				resultFileDiff.Hunks = append(resultFileDiff.Hunks, comparedFileDiff)
			}
			i++
			j++
		}
	}
	return &resultFileDiff
}

func compareHunk(oldHunk, newHunk *diff.Hunk) *diff.Hunk {
	// diffs are the same
	if (oldHunk.OrigStartLine == newHunk.OrigStartLine) &&
		(oldHunk.OrigLines == newHunk.OrigLines) &&
		(oldHunk.OrigNoNewlineAt == newHunk.OrigNoNewlineAt) &&
		(bytes.Equal(oldHunk.Body, newHunk.Body)) {
		return nil
	}

	revertedOldHunk := revertedHunkBody(oldHunk)
	return &diff.Hunk{OrigStartLine: oldHunk.NewStartLine,
		OrigLines:       oldHunk.NewLines,
		OrigNoNewlineAt: oldHunk.OrigNoNewlineAt,
		NewStartLine:    newHunk.NewStartLine,
		NewLines:        newHunk.NewLines,
		Section:         oldHunk.Section + newHunk.Section,
		// TODO: check the start position here
		StartPosition: oldHunk.StartPosition,
		Body:          append(revertedOldHunk.Body, newHunk.Body...),
	}
}

func revertedHunkBody(hunk *diff.Hunk) *diff.Hunk {
	var newBody []string

	lines := strings.Split(string(hunk.Body), "\n")

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+"):
			newBody = append(newBody, "-"+line[1:])
		case strings.HasPrefix(line, "-"):
			newBody = append(newBody, "+"+line[1:])
		default:
			newBody = append(newBody, line)
		}
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
