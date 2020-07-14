package interdiff

import (
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"io"
)

// RollupDiff does things
func RollupDiff(oldSource, newSource, diff io.Reader) (string, error) {
	return "", nil
}

// RollupDiffPath does things
func RollupDiffPath(oldSourcePath, newSourcePath string, diff io.Reader) (string, error) {
	return "", nil
}

// InterDiff does things
// TODO: source don't needed?
func InterDiff(source, oldDiff, newDiff io.Reader) (string, error) {
	oldFileDiffs, errOldFiles := diff.NewMultiFileDiffReader(oldDiff).ReadAllFiles()
	newFileDiffs, errNewFiles := diff.NewMultiFileDiffReader(newDiff).ReadAllFiles()

	if errOldFiles == nil {
		fmt.Errorf("error parsing old diffs")
		return "", errOldFiles
	}

	if errNewFiles == nil {
		fmt.Errorf("error parsing new diffs")
		return "", errNewFiles
	}

	result := ""

	// TODO: arrays need to be sorted by filenames of origin
	i, j := 0, 0
	for (i < len(oldFileDiffs)) && (j < len(newFileDiffs)){
		switch {
			case oldFileDiffs[i].OrigName == newFileDiffs[j].OrigName:
				result += compareFileDiff(oldFileDiffs[i], newFileDiffs[j]).String()
				i++
				j++
			case oldFileDiffs[i].OrigName < newFileDiffs[j].OrigName:
				// TODO: paste parent directory
				result += fmt.Sprintf("Only in <parent_dir>: %s", oldFileDiffs[i].NewName)
				i++
			default:
				// TODO: paste parent directory
				result += fmt.Sprintf("Only in <parent_dir>: %s", newFileDiffs[j].NewName)
				j++
		}
	}

	return result, nil
}

// InterDiffPath does things
func InterDiffPath(sourcePath string, oldDiff, newDiff io.Reader) (string, error) {
	return "", nil
}


func compareFileDiff(oldFileDiff, newFileDiff *diff.FileDiff) *diff.FileDiff{
	// TODO: something with extendent (extended header lines)
	resultFileDiff := diff.FileDiff{oldFileDiff.NewName, oldFileDiff.NewTime,
		newFileDiff.NewName, newFileDiff.NewTime, nil, []*diff.Hunk}

	i, j := 0, 0
	for (i < len(oldFileDiff.Hunks)) && (j < len(newFileDiff.Hunks)){
		switch {
		case oldFileDiff.Hunks[i].OrigStartLine + oldFileDiff.Hunks[i].OrigLines < newFileDiff.Hunks[j].OrigStartLine:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks,
				revertedHunkBody(oldFileDiff.Hunks[i]))
			i++
		case newFileDiff.Hunks[i].OrigStartLine + newFileDiff.Hunks[i].OrigLines < oldFileDiff.Hunks[j].OrigStartLine:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks,
				revertedHunkBody(newFileDiff.Hunks[j]))
			j++
		// Hunks are overlapping
		default:
			resultFileDiff.Hunks = append(resultFileDiff.Hunks,
				compareHunk(oldFileDiff.Hunks[i], newFileDiff.Hunks[j]))
			i++
			j++
		}
	}
	return &resultFileDiff
}

// TODO:
func compareHunk(oldHunk, newHunk *diff.Hunk) *diff.Hunk{
	return &diff.Hunk{}
}

// TODO:
func revertedHunkBody(hunk *diff.Hunk) *diff.Hunk{
	return &diff.Hunk{}
}