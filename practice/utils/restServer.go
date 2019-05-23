package utils

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"io"
	"net/http"

)

func main() {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(hello))
	restful.Add(ws) // ws被添加到默认的container restful.DefaultContainer中
	fmt.Print("Server starting on port 3001\n")
	http.ListenAndServe(":3001", nil)
}

func hello(req *restful.Request, resp *restful.Response) {
	io.WriteString(resp, "hello")
}

type Article struct {
	Name string
	Body string
}
