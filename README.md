# gocover

## Overview

`gocover` is a go unit test coverage explorer and inspector, providing go module level test coverage based on `go test` coverage result, as well as diff coverage between git commits. Plus, the tool supports annotations of ignoring file/[blocks](https://go.dev/blog/cover) at coverage calculation stage.

![project overview](./docs/images/overview.svg)

## Installation

### Install From Source

- Clone the repo
- Build Binary
```bash
go build .
```

## Usage

### Run Coverage Check

- Run test and get `coverage.out`
```bash
go test -coverprofile=coverage.out
```
- Get diff coverage
```bash
gocover --cover-profile=coverage.out --compare-branch=origin/master 
```
- Get overall coverage
```bash
gocover full --cover-profile=coverage.out
```
- Check the coverage detail at `coverage.html`

### Set Ignore Annotations

#### Ignore files

Put `//+gocover:ignore:file` at any line in a file to ignore a file at coverage inspection. Note that `//+gocover:ignore:file` has the highest priority, it will overrides other ignoreing annotation.

```go
//+gocover:ignore:file
package foo
func foo() {}
```

#### Ignore Block

We follow the definition of [basic block](https://go.dev/blog/cover) from `go test` to keep the same logic on coverage calculation.

Note that it is different from the [golang block](https://go.dev/ref/spec#Blocks). If you are not sure about the definition of the block, you can check the detail about every `block` within your change at the `coverage.out` file. Make sure to put the annotation into the `block`.

```go
package main

import "fmt"

var i, j int = 1, 2

func case1() { //+gocover:ignore:block           -|
 var c, python, java = true, false, "no!"      // | -> Block ignored
 fmt.Println(i, j, c, python, java)            //-|
}

func case2(x int) {//+gocover:ignore:block       -|
 var c, python, java = true, false, "no!"      // | -> Block ignored
 if x > 0 {                                    //-|
  fmt.Println(i, j, c, python, java)
 }

 fmt.Println(i, j, c, python, java, x)
}

func case3(x int) {//+gocover:ignore:block       -|
 var c, python, java = true, false, "no!"      // | -> Block1 ingored
 if x > 0 { //+gocover:ignore:block              -|
  fmt.Println(i, j, c, python, java)           // | -> Block2 ingored
 }                                             //-|

 fmt.Println(i, j, c, python, java, x)
}
```

### Get Coverage

Here is how we inspect the coverage:

- **Total Lines:** # of total lines of your change or the entire repo/module.
- **Ignored Lines:** # of the lines you ignored.
- **Effictive Lines:** total lines - ignored lines
- **Covered Lines:** # of the lines covered by test
- **Coverage:** Covered Lines / Effictive Lines

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft 
trademarks or logos is subject to and must follow 
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
