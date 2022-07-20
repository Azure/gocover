// Package annotation provides the utils for filtering.
//
// There are two kinds of ignore.
// 1. Ignore the whole go file.
//    `//+gocover:ignore:file`
// 2. Ignore a go code block.
//    `//+gocover:ignore:block`
//
//   Code block concept comes from the go coverage profile, the detail can be found at
//   https://cs.opensource.google/go/x/tools/+/master:cover/profile.go;drc=81efdbcac4736176ac97c60577b0069f76414c44;l=28
//   https://go.dev/ref/spec#Blocks gives more details about it.
//   For example:
//       pf, err := os.Open(fileName)
//       if err != nil {               -|
// 	         return nil, err            | -> code block
//       }                             -|
//
//       {
//           fmt.Println()                                      -|
//           profile, err := parseIgnoreProfilesFromReader(pf)   | -> code block
//           profile.Filename = fileName                        -|
//       }
package annotation
