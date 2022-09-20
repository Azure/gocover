package annotation

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/cover"
)

func TestIgnoreRegexp(t *testing.T) {
	t.Run("validate IgnoreRegexp", func(t *testing.T) {
		var testSuites = []struct {
			input  string
			expect []string
		}{
			{input: "//+gocover:ignore:file don't want test this block", expect: []string{"//+gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "// +gocover:ignore:file don't want test this block", expect: []string{"// +gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//    +gocover:ignore:file don't want test this block", expect: []string{"//    +gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//	+gocover:ignore:file don't want test this block", expect: []string{"//	+gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//+gocover:ignore:file some comments", expect: []string{"//+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "    //+gocover:ignore:file some comments", expect: []string{"    //+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "	//+gocover:ignore:file some comments", expect: []string{"	//+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "//+gocover:ignore:block some comments", expect: []string{"//+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "    //+gocover:ignore:block some comments", expect: []string{"    //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "	//+gocover:ignore:block some comments", expect: []string{"	//+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "  {  //+gocover:ignore:block some comments", expect: []string{"  {  //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "  //  //+gocover:ignore:block some comments", expect: []string{"  //  //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "//+gocover:ignore:file ", expect: []string{"//+gocover:ignore:file ", "file", " ", ""}},
			{input: "//+gocover:ignore:file  ", expect: []string{"//+gocover:ignore:file  ", "file", "  ", ""}},
			{input: "//+gocover:ignore:file", expect: []string{"//+gocover:ignore:file", "file", "", ""}},
			{input: "//+gocover:ignore:block", expect: []string{"//+gocover:ignore:block", "block", "", ""}},
			{input: "// +gocover:ignore:block", expect: []string{"// +gocover:ignore:block", "block", "", ""}},
			{input: "//+gocover:ignore:file: ", expect: []string{"//+gocover:ignore:file: ", "file", "", ": "}},
			{input: "//+gocover:ignore:file: comments", expect: []string{"//+gocover:ignore:file: comments", "file", "", ": comments"}},
			{input: "//+gocover:ignore:abc", expect: nil},
			{input: "//+gocover:ignore:123", expect: nil},
			{input: "//+gocover:ignore:", expect: nil},
		}

		for _, testSuite := range testSuites {
			match := IgnoreRegexp.FindStringSubmatch(testSuite.input)
			if len(match) != len(testSuite.expect) {
				t.Errorf("for input '%s', expect %d items, but get %d", testSuite.input, len(testSuite.expect), len(match))
			}
			n := len(match)
			for i := 0; i < n; i++ {
				if match[i] != testSuite.expect[i] {
					t.Errorf("for input '%s', expect item %d %s, but %s", testSuite.input, i, testSuite.expect[i], match[i])
				}
			}
		}
	})
}

func TestParseIgnoreAnnotation(t *testing.T) {
	t.Run("parse ignore annotation", func(t *testing.T) {
		testSuites := []struct {
			input    string
			kind     string
			comments string
			err      error
		}{
			{input: "//+gocover:ignore:file ignore this file!", kind: "file", comments: "ignore this file!", err: nil},
			{input: "//+gocover:ignore:file  ignore this file! ", kind: "file", comments: "ignore this file!", err: nil},
			{input: "{ //+gocover:ignore:block ignore this block!", kind: "block", comments: "ignore this block!", err: nil},
			{input: "{ //+gocover:ignore:block  ignore this block! ", kind: "block", comments: "ignore this block!", err: nil},
			{input: "//+gocover:ignore:abc ignore this block! ", kind: "", comments: "", err: nil},
			{input: "//+gocover:ignore:file  ", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:file", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block  ", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block:", kind: "", comments: "", err: ErrCommentsRequired},
		}

		for _, testCase := range testSuites {
			kind, comments, err := parseIgnoreAnnotation(testCase.input, 10)
			if kind != testCase.kind {
				t.Errorf("[%s] expect kind %s, but get %s", testCase.input, testCase.kind, kind)
			}
			if comments != testCase.comments {
				t.Errorf("[%s] expect comments %s, but get %s", testCase.input, testCase.comments, comments)
			}

			if testCase.err == nil && testCase.err != err {
				t.Errorf("[%s] expect error %s, but get %s", testCase.input, testCase.err, err)
			}
			if testCase.err != nil && err == nil {
				t.Errorf("[%s] expect error %s, but get %s", testCase.input, testCase.err, err)
			}
		}
	})
}

func TestParseIgnoreProfiles(t *testing.T) {

	findProfile := func(filename string, profiles []*cover.Profile) *cover.Profile {
		for _, p := range profiles {
			if filename == p.FileName {
				return p
			}
		}
		return nil
	}

	findIgnoreProfile := func(comment string, ignoreProfile *IgnoreProfile) (*cover.ProfileBlock, *IgnoreBlock) {
		for k, v := range ignoreProfile.IgnoreBlocks {
			if v.Comments == comment {
				return &k, v
			}
		}
		return nil, nil
	}

	t.Run("ignore block annotation", func(t *testing.T) {
		assertion := assert.New(t)

		coverFile := "testgo/coverprofile.out"
		testGoFile := "testgo/block.go"

		profiles, err := cover.ParseProfiles(filepath.Join("./testdata", coverFile))
		assertion.Nilf(err, "read cover profile file %s", filepath.Join("./testdata", coverFile))

		profile := findProfile(testGoFile, profiles)
		assertion.NotNilf(profile, "look cover profile for %s", testGoFile)

		ignoreProfile, err := ParseIgnoreProfiles(filepath.Join("./testdata", testGoFile), profile)
		assertion.Nilf(err, "ParseIgnoreProfiles")
		assertion.NotNilf(ignoreProfile, "ignoreprofile")

		// Refer to ./testdata/testgo/block.go and ./testdata/testgo/coverprofile.out
		testCases := []struct {
			blockIndex         string
			startLine, endLine int
		}{
			{blockIndex: "01", startLine: 8, endLine: 10},
			{blockIndex: "02", startLine: 12, endLine: 14},
			{blockIndex: "11", startLine: 17, endLine: 23},
			{blockIndex: "21", startLine: 25, endLine: 28},
			{blockIndex: "41", startLine: 37, endLine: 42},
			{blockIndex: "51", startLine: 61, endLine: 61},
			{blockIndex: "61", startLine: 73, endLine: 77},
			{blockIndex: "71", startLine: 80, endLine: 87},
			{blockIndex: "81", startLine: 100, endLine: 103},
			{blockIndex: "91", startLine: 113, endLine: 119},
			{blockIndex: "101", startLine: 129, endLine: 136},
			{blockIndex: "111", startLine: 142, endLine: 147},
			{blockIndex: "112", startLine: 147, endLine: 154},
			{blockIndex: "113", startLine: 157, endLine: 157},
			{blockIndex: "121", startLine: 161, endLine: 165},
			{blockIndex: "131", startLine: 169, endLine: 171},
			{blockIndex: "132", startLine: 171, endLine: 174},
		}

		for _, testCase := range testCases {
			blockIdentifier := fmt.Sprintf("ignore block %s", testCase.blockIndex)
			profileBlock, ignoreBlock := findIgnoreProfile(blockIdentifier, ignoreProfile)
			assertion.NotNilf(profileBlock, "profile block on %s", blockIdentifier)
			assertion.NotNilf(ignoreBlock, "ignore block on %s", blockIdentifier)
			assertion.Equalf(testCase.startLine, profileBlock.StartLine, "start line on %s", blockIdentifier)
			assertion.Equalf(testCase.endLine, profileBlock.EndLine, "end line on %s", blockIdentifier)
		}
	})

	t.Run("read file error", func(t *testing.T) {
		assertion := assert.New(t)
		_, err := ParseIgnoreProfiles("/nonexist", nil)
		assertion.Errorf(err, "when file not exist")
	})

	t.Run("ignore file annotation", func(t *testing.T) {
		assertion := assert.New(t)
		coverFile := "testgo/coverprofile.out"
		testGoFile := "testgo/main.go"

		profiles, err := cover.ParseProfiles(filepath.Join("./testdata", coverFile))
		assertion.Nilf(err, "read cover profile file %s", filepath.Join("./testdata", coverFile))

		profile := findProfile(testGoFile, profiles)
		assertion.NotNilf(profile, "look cover profile for %s", testGoFile)

		ignoreProfile, err := ParseIgnoreProfiles(filepath.Join("./testdata", testGoFile), profile)
		assertion.Nilf(err, "ParseIgnoreProfiles")
		assertion.NotNilf(ignoreProfile, "ignoreprofile")

		// refer to ./testdata/testgo/main.go
		assertion.Equal(ignoreProfile.Type, FILE_IGNORE)
		assertion.Equal(ignoreProfile.Comments, "ignore this file")
		assertion.Equal(ignoreProfile.Annotation, "//+gocover:ignore:file ignore this file")
	})

}
