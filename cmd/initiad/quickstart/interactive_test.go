package quickstart

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPromptChoice(t *testing.T) {
	input := "2\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result, err := promptChoice(reader, cmd, "Pick one", []string{"a", "b", "c"})
	require.NoError(t, err)
	require.Equal(t, "b", result)
}

func TestPromptChoice_InvalidInput(t *testing.T) {
	input := "abc\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	_, err := promptChoice(reader, cmd, "Pick one", []string{"a", "b", "c"})
	require.Error(t, err)
}

func TestPromptChoice_OutOfRange(t *testing.T) {
	input := "5\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	_, err := promptChoice(reader, cmd, "Pick one", []string{"a", "b", "c"})
	require.Error(t, err)
}

func TestPromptString_WithDefault(t *testing.T) {
	input := "\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result, err := promptString(reader, cmd, "Enter value", "mydefault")
	require.NoError(t, err)
	require.Equal(t, "mydefault", result)
}

func TestPromptString_WithInput(t *testing.T) {
	input := "custom_value\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result, err := promptString(reader, cmd, "Enter value", "mydefault")
	require.NoError(t, err)
	require.Equal(t, "custom_value", result)
}

func TestPromptYesNo_Yes(t *testing.T) {
	input := "y\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result, err := promptYesNo(reader, cmd, "Continue?")
	require.NoError(t, err)
	require.True(t, result)
}

func TestPromptYesNo_No(t *testing.T) {
	input := "n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result, err := promptYesNo(reader, cmd, "Continue?")
	require.NoError(t, err)
	require.False(t, result)
}

func TestPromptChoiceWithLabels(t *testing.T) {
	input := "1\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	choices := []choiceOption{
		{Value: "val1", Label: "Label One"},
		{Value: "val2", Label: "Label Two"},
		{Value: "val3", Label: "Label Three"},
	}

	result, err := promptChoiceWithLabels(reader, cmd, "Pick one", choices)
	require.NoError(t, err)
	require.Equal(t, "val1", result)
}
