package graph

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/v2/graph/llamago"
	"os"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestExampleMessageGraph(t *testing.T) {
	var llamagoModel string
	if llamagoModel = os.Getenv("LLAMAGO_TEST_MODEL"); llamagoModel == "" {
		t.Skip("LLAMAGO_TEST_MODEL not set")
		return
	}
	model, err := llamago.New(llamago.WithModel(os.Getenv(llamagoModel)))
	if err != nil {
		fmt.Println(err)
		return
	}

	g := NewGraph()

	g.AddNode("oracle", func(ctx context.Context, name string, state State) (State, error) {
		if state == nil {
			return nil, errors.New("No state")
		}
		r, err := model.GenerateContent(ctx, state["messages"].([]llms.MessageContent), llms.WithTemperature(0.0))
		if err != nil {
			return nil, err
		}
		state["output"] = llms.TextParts(llms.ChatMessageTypeAI, r.Choices[0].Content)
		return state, nil
	})

	g.AddNode(END, func(_ context.Context, name string, state State) (State, error) {
		return state, nil
	})

	g.AddEdge("oracle", END)
	g.SetEntryPoint("oracle")

	runnable, err := g.Compile()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// Let's run it!
	res, err := runnable.Invoke(ctx, State{"messages": []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "What is 1 + 1?"),
	}})
	if err != nil {
		panic(err)
	}

	fmt.Println(res)

	// Output:
	// [{human [What is 1 + 1?]} {ai [1 + 1 equals 2.]}]
}

func TestMessageGraph(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		buildGraph     func() *Graph
		inputMessages  State
		expectedOutput State
		expectedError  error
	}{
		{
			name: "Simple graph",
			buildGraph: func() *Graph {
				g := NewGraph()
				g.AddNode("node1", func(_ context.Context, name string, state State) (State, error) {
					state["messages"] = append(state["messages"].([]llms.MessageContent), llms.TextParts(llms.ChatMessageTypeAI, "Node 1"))
					return state, nil
				})
				g.AddNode("node2", func(_ context.Context, name string, state State) (State, error) {
					state["messages"] = append(state["messages"].([]llms.MessageContent), llms.TextParts(llms.ChatMessageTypeAI, "Node 2"))
					return state, nil
				})
				g.AddEdge("node1", "node2")
				g.AddEdge("node2", END)
				g.SetEntryPoint("node1")
				return g
			},
			inputMessages: State{"messages": []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "Input")}},
			expectedOutput: State{"messages": []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeHuman, "Input"),
				llms.TextParts(llms.ChatMessageTypeAI, "Node 1"),
				llms.TextParts(llms.ChatMessageTypeAI, "Node 2"),
			}},
			expectedError: nil,
		},
		{
			name: "Entry point not set",
			buildGraph: func() *Graph {
				g := NewGraph()
				g.AddNode("node1", func(_ context.Context, name string, state State) (State, error) {
					return state, nil
				})
				return g
			},
			expectedError: ErrEntryPointNotSet,
		},
		{
			name: "Node not found",
			buildGraph: func() *Graph {
				g := NewGraph()
				g.AddNode("node1", func(_ context.Context, name string, state State) (State, error) {
					return state, nil
				})
				g.AddEdge("node1", "node2")
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: fmt.Errorf("%w: node2", ErrNodeNotFound),
		},
		{
			name: "No outgoing edge",
			buildGraph: func() *Graph {
				g := NewGraph()
				g.AddNode("node1", func(_ context.Context, name string, state State) (State, error) {
					return state, nil
				})
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: fmt.Errorf("%w: node1", ErrNoOutgoingEdge),
		},
		{
			name: "Error in node function",
			buildGraph: func() *Graph {
				g := NewGraph()
				g.AddNode("node1", func(_ context.Context, name string, _ State) (State, error) {
					return nil, errors.New("node error")
				})
				g.AddEdge("node1", END)
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: errors.New("error in node node1: node error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := tc.buildGraph()
			runnable, err := g.Compile()
			if err != nil {
				if tc.expectedError == nil || !errors.Is(err, tc.expectedError) {
					t.Fatalf("unexpected compile error: %v", err)
				}
				return
			}

			output, err := runnable.Invoke(context.Background(), tc.inputMessages)
			if err != nil {
				if tc.expectedError == nil || err.Error() != tc.expectedError.Error() {
					t.Fatalf("unexpected invoke error: '%v', expected '%v'", err, tc.expectedError)
				}
				return
			}

			if tc.expectedError != nil {
				t.Fatalf("expected error %v, but got nil", tc.expectedError)
			}

			if len(output) != len(tc.expectedOutput) {
				t.Fatalf("expected output length %d, but got %d", len(tc.expectedOutput), len(output))
			}
			if tc.expectedOutput == nil && output == nil {
				return
			}
		})
	}
}

func TestConditionalEdge(t *testing.T) {
	g := NewGraph()

	g.AddNode("node_0", func(_ context.Context, name string, state State) (State, error) {
		text := "I am node 0"
		t.Log(text)
		state["node_0"] = text
		return state, nil
	})
	g.AddNode("node_1", func(_ context.Context, name string, state State) (State, error) {
		text := "I am node 1"
		t.Log(text)
		state["node_1"] = text
		return state, nil
	})
	g.AddNode("node_2", func(_ context.Context, name string, state State) (State, error) {
		text := "I am node 2"
		t.Log(text)
		state["node_2"] = text
		return state, nil
	})
	g.AddNode("node_3", func(_ context.Context, name string, state State) (State, error) {
		text := "I am node 3"
		t.Log(text)
		state["node_3"] = text
		return state, nil
	})
	g.AddNode(END, func(_ context.Context, name string, state State) (State, error) {
		t.Log("I am end")
		return state, nil
	})

	g.AddEdge("node_0", "node_1")
	g.AddConditionalEdge("node_1", func(_ context.Context, name string, state State) string {
		if state["input"] == "node 2 branch" {
			return "node_2"
		}
		return "node_3"
	})

	g.AddEdge("node_2", END)
	g.AddEdge("node_3", END)

	g.SetEntryPoint("node_0")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	t.Run("node_2 branch", func(t *testing.T) {
		input := State{"input": "node 2 branch"}
		output, err := runnable.Invoke(context.Background(), input)
		if err != nil {
			t.Fatalf("invoke error: %v", err)
		}
		if _, ok := output["node_2"]; !ok {
			t.Errorf("expected node_2 branch, got %v", output)
		}
		t.Log(output)
	})

	t.Run("node_3 branch", func(t *testing.T) {
		input := State{}
		output, err := runnable.Invoke(context.Background(), input)
		if err != nil {
			t.Fatalf("invoke error: %v", err)
		}
		if _, ok := output["node_3"]; !ok {
			t.Errorf("expected node_3 branch, got %v", output)
		}
		t.Log(output)
	})
}
