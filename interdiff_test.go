package interdiff

import (
	"bytes"
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

// Reference: https://www.programming-books.io/essential/go/normalize-newlines-1d3abcf6f17c4186bb9617fa14074e48
// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
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

			var readerA io.Reader = fileA
			var readerB io.Reader = fileB

			currentResult, err := InterDiff(readerA, readerB)
			if err != nil {
				t.Error(err)
			}

			if !bytes.Equal(NormalizeNewlines([]byte(currentResult)), NormalizeNewlines(correctResult)) {
				t.Errorf("File contents mismatch for %s.\nExpected:\n%x\nGot:\n%x\n",
					tt.resultFile, correctResult, currentResult)
			}
		})
	}
}
