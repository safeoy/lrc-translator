package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

var ErrTooManyRequests = errors.New("too many requests")

var client *openai.Client

type TranslationRequest struct {
	Text     string `json:"text"`
	Model    string `json:"model"`
	Language string `json:"target_language"`
}

type TranslationResponse struct {
	Translations []string `json:"translations"`
}

func main() {
	inputFile := flag.String("input", "", "Input LRC file")
	outputFile := flag.String("output", "", "Output LRC file")
	language := flag.String("language", "en", "Translation language")
	apiKey := flag.String("apikey", "", "OpenAI API key")

	flag.Parse()

	if *inputFile == "" || *outputFile == "" || *apiKey == "" {
		fmt.Println("Please specify input file, output file, and OpenAI API key")
		return
	}

	client = openai.NewClient(*apiKey)

	// Open input file
	input, err := os.Open(*inputFile)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return
	}
	defer input.Close()

	// Open output file
	output, err := os.Create(*outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer output.Close()

	// Read input file line by line
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line is a valid lyrics line (contains brackets [])
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			// Split the line into timestamp and lyrics
			parts := strings.SplitN(line, "]", 2)
			if len(parts) == 2 {
				timestamp := parts[0] + "]"
				lyrics := parts[1]

				tranlyrics, err := translateLyrics(lyrics, *language)
				time.Sleep(30 * time.Second)
				if err != nil {
					fmt.Println("Error translateLyrics:", err)
					return
				}

				// Modify the lyrics by adding "abc" at the beginning and "123" at the end
				lyrics = tranlyrics + "|" + lyrics

				// Write the modified line (timestamp + modified lyrics) to the output file
				_, err = fmt.Fprintln(output, timestamp+lyrics)
				if err != nil {
					fmt.Println("Error writing to output file:", err)
					return
				}
			}
		} else {
			// Write non-lyrics lines (e.g., metadata) as is to the output file
			_, err := fmt.Fprintln(output, line)
			if err != nil {
				fmt.Println("Error writing to output file:", err)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input file:", err)
		return
	}

	fmt.Println("Lyrics processed successfully!")
}

func translateLyrics(src, lang string) (string, error) {
	msg := openai.ChatCompletionMessage{
		Role: "user", Content: fmt.Sprintf("Translate \"%s\" to %s. Give the result directly. Don't explain. Don't quote output.", src, lang),
	}

	request := openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{
			msg,
		},
		MaxTokens:   1024,
		Stop:        []string{"STOP"},
		Temperature: 0.1,
	}

	resp, err := client.CreateChatCompletion(context.Background(), request)
	if err != nil {
		return "", err
	}

	result := ""
	if len(resp.Choices) > 0 {
		result = strings.TrimSpace(resp.Choices[0].Message.Content)
	}
	return result, err
}
