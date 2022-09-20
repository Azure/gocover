package main

import "fmt"

var i, j int = 1, 2

// +gocover:ignore:block ignore block 01
func testCase0() {
	fmt.Println("case 0")
	if i > j {
		fmt.Println(i)
	} else { // +gocover:ignore:block ignore block 02
		fmt.Println(j)
	}
}

func testCase1() {

	//+gocover:ignore:block ignore block 11

	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func testCase2() { //+gocover:ignore:block ignore block 21
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func testCase3() {
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)

	//+gocover:ignore:block ignore block 31
}

func testCase4(x int) { //+gocover:ignore:block ignore block 41

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}

	fmt.Println(i, j, c, python, java, x)
}

func testCase5(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}

	//+gocover:ignore:block ignore block 51
	// ignore block 5 should include following print line

	fmt.Println(i, j, c, python, java, x)
}

func testCase6(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}

	fmt.Println("print results")
	//+gocover:ignore:block ignore block 61
	// ignore block 5 should include following print line

	fmt.Println(i, j, c, python, java, x)
}

func testCase7(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	//+gocover:ignore:block ignore block 71

	if x > 0 {
		fmt.Println()
		fmt.Println(i, j, c, python, java)
	}

	fmt.Println(i, j, c, python, java, x)
}

func testCase8(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 { //+gocover:ignore:block ignore block 81
		fmt.Println()
		fmt.Println(i, j, c, python, java)
	}

	fmt.Println(i, j, c, python, java, x)
}

func testCase9(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {
		//+gocover:ignore:block ignore block 91

		fmt.Println()
		fmt.Println(i, j, c, python, java)

	}

	fmt.Println(i, j, c, python, java, x)
}

func testCase10(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {

		fmt.Println()
		fmt.Println(i, j, c, python, java)

		//+gocover:ignore:block ignore block 101

	}

	fmt.Println(i, j, c, python, java, x)
}

// +gocover:ignore:block ignore block 111
func testCase11(x int) {

	// declare variables
	var c, python, java = true, false, "no!"

	if x > 0 {

		fmt.Println()
		fmt.Println(i, j, c, python, java)

		//+gocover:ignore:block ignore block 112

	}

	//+gocover:ignore:block ignore block 113
	fmt.Println(i, j, c, python, java, x)
}

func testCase12(func() int) {
	{ //+gocover:ignore:block ignore block 121
		fmt.Printf("A")
		fmt.Printf("A")
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

func case13(x int) {
	//+gocover:ignore:block ignore block 131
	testCase12(func() int {
		//+gocover:ignore:block ignore block 132
		return 1
	})
}

//+gocover:ignore:block ignore block 142
//+gocover:ignore:block ignore block 143
