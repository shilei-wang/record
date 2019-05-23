package utils


import (
	"fmt"
)

func main() {
	chanCap := 5
	ch := make(chan int, chanCap) //创建channel，容量为5
	//j := 9
	ch2 := make(chan int, chanCap)
	//ch2 <- j
	for i := 0; i < chanCap; i++ { //通过for循环，向channel里填满数据
		select { //通过select随机的向channel里追加数据
		case ch <- 1:
		case ch <- 2:
		case ch <- 3:
		case k:= <- ch2:
			fmt.Printf("9 is recieved : %d\n", k)
		}
	}
	for i := 0; i < chanCap; i++ {
		fmt.Printf("%v\n", <-ch)
	}
}