package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/HiroCloud/llm-client/llm_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStructDef(t *testing.T) {
	d, err := CreateDef(llm_models.Message{})
	require.NoError(t, err)

	mcpTool, err := GetToolMCP(d)
	require.NoError(t, err)

	f, err := json.MarshalIndent(mcpTool, "", "  ")
	require.NoError(t, err)
	fmt.Println(string(f))

	f, err = json.MarshalIndent(d, "", "  ")
	require.NoError(t, err)
	fmt.Println(string(f))
}

func TestCreateDef(t *testing.T) {
	type TestCase struct {
		Name           string
		Func           interface{}
		ExpectedErr    error
		args           []interface{}
		expectedOutput []interface{}
	}

	tcs := []TestCase{
		{
			Name:           "Normal",
			Func:           PrintTest,
			ExpectedErr:    nil,
			args:           []interface{}{"a", 1},
			expectedOutput: []interface{}{"a 1"},
		},
		{
			Name:           "Fail Params",
			Func:           PrintTest,
			ExpectedErr:    errors.New("expected 2 arguments, got 1"),
			args:           []interface{}{"a"},
			expectedOutput: []interface{}{"a 1"},
		},
		{
			Name:           "No Context",
			Func:           T,
			args:           []interface{}{"a", 1},
			expectedOutput: []interface{}{"a 1 T"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			d, err := CreateDef(tc.Func)
			require.NoError(t, err)

			resp, err := CallTool(context.Background(), d, tc.args...)
			if tc.ExpectedErr != nil {
				assert.Error(t, err, tc.ExpectedErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectedOutput, resp)
			output, err := SaveTool("toolsjson/", "", d)
			require.NoError(t, err)
			assert.Equal(t, output, "toolsjson/"+d.Function.Name+".json")
			_, err = NewToolFromFile(tc.Func, output)
			require.NoError(t, err)
		})
	}
}

// PrintTest is a test function to generate a list of strings
func PrintTest(ctx context.Context, name string, amount int) string {
	return fmt.Sprintf("%s %d", name, amount)
}

func T(name string, amount int) string {
	return fmt.Sprintf("%s %d T", name, amount)
}
