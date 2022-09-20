### How to generate test data

1. Create a go module called `testgo`
2. Create two files, `block.go` and `main.go`.
3. Execute command `go test ./... -coverprofile coverprofile.out`.

### How to contribute to test data
1. Append your test case functions in the `block.go` testdata file.
2. Generate `coverprofile.out` according to the section `How to generate test data`.
3. Update the `coverprofile.out` testdata file with the new coverprofile.out generated at step 2.
4. Add test case in `ignore block annotation` test suite for `TestParseIgnoreProfiles` function based on the `coverprofile.out`.

Note: Please add new cases and do not modify them, because changes would generate different profile block,
which would modify the test cases correspondingly manually.
