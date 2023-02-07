package main
import "github.com/mavenraven/i-spy/parser"
import "fmt"

func main() {
    parser.ParseSingleFile("hello")
    fmt.Println("hello world")
}
