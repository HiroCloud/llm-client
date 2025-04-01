## LLM Client

### Features
- Generate Tool Calls based of structs
    - Based of comments
- Generate Tool Calls based of functions
    - Based of comments
- Call Tools dynamically
- MCP support, based on [MCP-GO](https://github.com/mark3labs/mcp-go)

### Install
```bash
go get github.com/HiroCloud/llm-client
```



### Generate

#### Struct

```go
package main

import (
  "fmt"
  "github.com/HiroCloud/llm-client/tools"
)

// Person a person struct
// Name of the person ie firstname lastname
// Age of the person
type Person struct {
  Name string
  Age  int
}

func main() {
  personTool, err := tools.CreateDef(Person{})
  if err != nil {
    panic(err)
  }
  output, err := tools.SaveTool("toolsjson/", "", personTool)
  if err != nil {
    panic(err)
  }
  fmt.Printf("saved tool: %s", output)
  
  // NOTE: Always load data from a file createDef only works when running locally
  _,_ = tools.NewToolFromFile(Person{},output)
}

```

```json
{
  "function": {
    "name": "Person",
    "description": "Person defines a person object",
    "param_order": [
      "Name",
      "Age"
    ],
    "parameters": {
      "type": "object",
      "properties": {
        "Age": {
          "type": "integer",
          "description": "n/a"
        }
      },
      "required": [
        "Name",
        "Age"
      ]
    }
  },
  "exit_func": false,
  "write_to_chat": ""
}
```

#### Function

```go
package main

import (
  "context"
  "encoding/json"
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
  tools.CallTool(context.Background(),personTool,"name")
  
  tools.CallTool(context.Background(),personToolCtx,"name2")
  
}

func GetPerson(name string) Person {
  return Person{}
}

func GetPersonCtx(ctx context.Context,name string) Person {
  return Person{}
}
```

```json
{
  "function": {
    "name": "main.GetPerson",
    "param_order": [
      "name"
    ],
    "parameters": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        }
      },
      "required": [
        "name"
      ]
    }
  },
  "exit_func": false,
  "write_to_chat": ""
}
```