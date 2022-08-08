package finalround

import "fmt"

func Case1(x int) { //+gocover:ignore:block
	fmt.Printf("A")
	if x > 0 { //+gocover:ignore:block
		fmt.Printf("A")
	} //+gocover:ignore:block
	fmt.Printf("A")
}

func Case2(x int) { //+gocover:ignore:block
	fmt.Printf("A")
	fmt.Printf("A")
	//+gocover:ignore:block
	fmt.Printf("A")
}

func Case3(x int) {
	{
		fmt.Printf("A")
		fmt.Printf("A")
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

func Case4(x int) {
	{
		//+gocover:ignore:block
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

//+gocover:ignore:block//+gocover:ignore:block
func Case5(x int) {
	{ //+gocover:ignore:block
		fmt.Printf("A")
		fmt.Printf("A")
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

func Case6(x int) { //+gocover:ignore:block
	{ //+gocover:ignore:block
		//+gocover:ignore:block
		fmt.Printf("A")
		fmt.Printf("A")
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

func Case7(x int) {
	{
		{
			//+gocover:ignore:block
			fmt.Printf("A")
		}
	}
	fmt.Printf("A")
}

//+gocover:ignore:block//+gocover:ignore:block
func Case8(func() int) {
	{ //+gocover:ignore:block
		fmt.Printf("A")
		fmt.Printf("A")
		fmt.Printf("A")
	}
	fmt.Printf("A")
}

func Case9(x int) {
	//+gocover:ignore:block
	Case8(func() int {
		return 1
	})
}
