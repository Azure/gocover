## How to prepare test data

In the root folder of the project, run the following command:

```bash
$ go test ./... -coverprofile cover.out -coverpkg=./... cover.out
$ mv cover.out pkg/parser/testdata/
```
