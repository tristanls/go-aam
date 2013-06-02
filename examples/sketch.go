package main

import (
  "fmt"
  "github.com/tristanls/go-aam"
)

func Change(event aam.Event) {
  self := event.Self()
  self.Send(self, event.Message)
  self.Send(self, event.Message)
  self.Become(Print)
  self.Send(self, event.Message)
  self.Send(self, event.Message)
}

func Print(event aam.Event) {
  for _, param := range event.Message {
    fmt.Println(param.(string))
  }
}

func main() {
  configuration := aam.New()
  configuration.Trace = true

  change := configuration.Create(Change)
  configuration.Send(change, aam.Message{"foo"})

  for configuration.HasEvents() {
    configuration.Dispatch()
  }
}