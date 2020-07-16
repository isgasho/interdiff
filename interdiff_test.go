package interdiff

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var interDiffFileTests = []struct {
	resultFile string
	diffAFile  string
	diffBFile  string
}{
	{"test_examples/result_1.txt", "test_examples/diff_1_a.txt", "test_examples/diff_1_b.txt"},
	{"test_examples/result_2.txt", "test_examples/diff_2_a.txt", "test_examples/diff_2_b.txt"},
}

func TestInterDiffMode(t *testing.T) {
	for _, tt := range interDiffFileTests {
		t.Run(tt.resultFile, func(t *testing.T) {
			var fileA, errA = os.Open(tt.diffAFile)
			var fileB, errB = os.Open(tt.diffBFile)

			if errA != nil {
				t.Errorf("Error in opening %s file.", tt.diffAFile)
			}

			if errB != nil {
				t.Errorf("Error in opening %s file.", tt.diffBFile)
			}

			correctResult, err := ioutil.ReadFile(tt.resultFile)

			if err != nil {
				t.Error(err)
			}

			correctResultStr := string(correctResult)

			var readerA io.Reader = fileA
			var readerB io.Reader = fileB

			currentResult, err := InterDiff(readerA, readerB)
			if err != nil {
				t.Error(err)
			}

			if currentResult != correctResultStr {
				t.Errorf("File contents mismatch for %s.\nExpected:\n%s\nGot:\n%s\n",
					tt.resultFile, correctResultStr, currentResult)
			}
		})
	}
}
