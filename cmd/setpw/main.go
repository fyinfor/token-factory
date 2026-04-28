package main
import (
  "fmt"
  "os"
  "golang.org/x/crypto/bcrypt"
)
func main() {
  h, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
  if err != nil { panic(err) }
  fmt.Println(string(h))
}
