package parser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/Azure/gocover/pkg/annotation"
)

type Package struct {
	// Name is the canonical path of the package.
	Name string

	// Functions is a list of functions registered with this package.
	Functions []*Function

	// IgnoreProfiles is a list of ignore profiles that within this package.
	IgnoreProfiles []*annotation.IgnoreProfile
}

type Function struct {
	// Name is the name of the function. If the function has a receiver, the
	// name will be of the form T.N, where T is the type and N is the name.
	Name string

	// File is the full path to the file in which the function is defined.
	File string

	// Start is the start offset of the function's signature.
	Start int

	// End is the end offset of the function.
	End int

	// StartLine is the start line number of the function.
	StartLine int

	// EndLine is the end line number of the function.
	EndLine int

	// statements registered with this function.
	Statements []*Statement
}

type Statement struct {
	// Start is the start offset of the statement.
	Start int

	// End is the end offset of the statement.
	End int

	// StartLine is the start line number of the statement.
	StartLine int

	// EndLine is the end line number of the statement.
	EndLine int

	// Reached is the number of times the statement was reached.
	Reached int64

	// State indicates whether current statement is changed or not.
	State State

	// Mode indicates whether current statement counts for coverage.
	Mode Mode
}

// State represents statement's state.
// "Original" means it hasn't changed compared with compare branch (master or main)
// "Changed" means it has been changed compared with compare branch (master or main)
type State string

// Mode represents statement's mode.
// "Keep" means it will be used in coverage calculation
// "Ignore" means it won't be used in coverage calculation
type Mode string

const (
	Original State = "Original"
	Changed  State = "Changed"

	Keep   Mode = "Keep"
	Ignore Mode = "Ignore"
)

// Accumulate will accumulate the coverage information from the provided
// Package into this Package.
func (p *Package) Accumulate(p2 *Package) error {
	if p.Name != p2.Name {
		return fmt.Errorf("names do not match: %q != %q", p.Name, p2.Name)
	}
	if len(p.Functions) != len(p2.Functions) {
		return fmt.Errorf("function counts do not match: %d != %d", len(p.Functions), len(p2.Functions))
	}
	for i, f := range p.Functions {
		err := f.Accumulate(p2.Functions[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// Accumulate will accumulate the coverage information from the provided
// Function into this Function.
func (f *Function) Accumulate(f2 *Function) error {
	if f.Name != f2.Name {
		return fmt.Errorf("names do not match: %q != %q", f.Name, f2.Name)
	}
	if f.File != f2.File {
		return fmt.Errorf("files do not match: %q != %q", f.File, f2.File)
	}
	if f.Start != f2.Start || f.End != f2.End {
		return fmt.Errorf("source ranges do not match: %d-%d != %d-%d", f.Start, f.End, f2.Start, f2.End)
	}
	if len(f.Statements) != len(f2.Statements) {
		return fmt.Errorf("number of statements do not match: %d != %d", len(f.Statements), len(f2.Statements))
	}
	for i, s := range f.Statements {
		err := s.Accumulate(f2.Statements[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// Accumulate will accumulate the coverage information from the provided
// Statement into this Statement.
func (s *Statement) Accumulate(s2 *Statement) error {
	if s.Start != s2.Start || s.End != s2.End {
		return fmt.Errorf("source ranges do not match: %d-%d != %d-%d", s.Start, s.End, s2.Start, s2.End)
	}
	s.Reached += s2.Reached
	return nil
}

// Packages represents a set of Package structures.
// The "AddPackage" method may be used to merge package
// coverage results into the set.
type Packages []*Package

// AddPackage adds a package's coverage information to the
func (ps *Packages) AddPackage(p *Package) {
	i := sort.Search(len(*ps), func(i int) bool {
		return (*ps)[i].Name >= p.Name
	})
	if i < len(*ps) && (*ps)[i].Name == p.Name {
		(*ps)[i].Accumulate(p)
	} else {
		head := (*ps)[:i]
		tail := append([]*Package{p}, (*ps)[i:]...)
		*ps = append(head, tail...)
	}
}

// ReadPackages takes a list of filenames and parses their
// contents as a Packages object.
//
// The special filename "-" may be used to indicate standard input.
// Duplicate filenames are ignored.
func ReadPackages(filenames []string) (ps Packages, err error) {
	copy_ := make([]string, len(filenames))
	copy(copy_, filenames)
	filenames = copy_
	sort.Strings(filenames)

	// Eliminate duplicates.
	unique := []string{filenames[0]}
	if len(filenames) > 1 {
		for _, f := range filenames[1:] {
			if f != unique[len(unique)-1] {
				unique = append(unique, f)
			}
		}
	}

	// Open files.
	var files []*os.File
	for _, f := range filenames {
		if f == "-" {
			files = append(files, os.Stdin)
		} else {
			file, err := os.Open(f)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			files = append(files, os.Stdin)
		}
	}

	// Parse the files, accumulate Packages.
	for _, file := range files {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		result := &struct{ Packages []*Package }{}
		err = json.Unmarshal(data, result)
		if err != nil {
			return nil, err
		}
		for _, p := range result.Packages {
			ps.AddPackage(p)
		}
	}
	return ps, nil
}
