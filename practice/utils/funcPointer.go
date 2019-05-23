package utils

import "fmt"
import "time"

func goFunc1(f func()) {
	f()
}

//空interface(interface{})不包含任何的method，正因为如此，所有的类型都实现了空interface。
//空interface对于描述起不到任何的作用(因为它不包含任何的method），但是空interface在我们
//需要存储任意类型的数值的时候相当有用，因为它可以存储任意类型的数值。它有点类似于C语言的void*类型。
func goFunc2(f func(interface{}), i interface{}) {
	f(i) // f2(i)
}

func goFunc(f interface{}, args... interface{}) {

	if len(args) > 1 {
		f.(func(...interface{}))(args)
	} else if len(args) == 1 {
		f.(func(interface{}))(args[0])
	} else {
		f.(func())()
	}

}

func f1() {
	fmt.Println("f1 done")
}

func f2(i interface{}) {
	fmt.Println("f2 done", i)
}

func f3(args... interface{}) {
	fmt.Println("f3 done", args)
}

func main() {
	goFunc1(f1)
	goFunc2(f2, 100)

	//下面太难理解
	goFunc(f1)
	goFunc(f2, "xxxx")
	goFunc(f3, "hello", "world", 1, 3.14)
	time.Sleep(5 * time.Second)
}
