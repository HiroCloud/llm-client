package main

import (
	"context"
	"fmt"
	"github.com/HiroCloud/llm-client/tools"
)

// Person a person struct
type Person struct {
	// name of the person ie firstname lastname
	Name string
	// age of the person
	Age int
}

func main() {
	personTool, err := tools.CreateDef(GetPerson)
	if err != nil {
		panic(err)
	}
	output, err := tools.SaveTool("toolsjson/", "", personTool)
	if err != nil {
		panic(err)
	}
	fmt.Printf("saved tool: %s", output)

	personToolCtx, err := tools.CreateDef(GetPersonCtx)
	if err != nil {
		panic(err)
	}
	tools.CallTool(context.Background(), personTool, "name")
	tools.CallTool(context.Background(), personToolCtx, "name2")
}

func GetPerson(name string) Person {
	return Person{}
}

func GetPersonCtx(ctx context.Context, name string) Person {
	return Person{}
}
