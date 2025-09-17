package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"unicode"
)

// version is set by the linker during the build process
var version = "dev"

// Job represents a batch of items to be processed.
type Job struct {
	ID        int
	JSONBytes []byte
	ItemData  []interface{} // A batch of items
}

// Result defines the structure for the output.
type Result struct {
	ID     int           `json:"-"` // Used for sorting, ignored in JSON output
	Input  []interface{} `json:"input"`
	Output string        `json:"output"`
	Error  error         `json:"-"` // Internal use, ignored in JSON output
}

// processItem executes the llm-cli command with a batch of items.
func processItem(jsonBytes []byte, systemPrompt string, profile string, stream bool) (string, error) {
	args := []string{"prompt", "--system-prompt", systemPrompt, "--user-prompt-file", "-"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	if stream {
		args = append(args, "--stream")
	}

	llmCmd := exec.Command("llm-cli", args...)

	if stream {
		llmCmd.Stdout = os.Stdout
		llmCmd.Stderr = os.Stderr
	} else {
		var outb, errb bytes.Buffer
		llmCmd.Stdout = &outb
		llmCmd.Stderr = &errb
	}

	llmStdin, err := llmCmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("error creating llm-cli stdin pipe: %w", err)
	}

	if err := llmCmd.Start(); err != nil {
		llmStdin.Close()
		return "", fmt.Errorf("error starting llm-cli: %w. Is llm-cli installed and in your PATH?", err)
	}

	go func() {
		defer llmStdin.Close()
		llmStdin.Write(jsonBytes)
	}()

	err = llmCmd.Wait()

	if stream {
		if err != nil {
			return "", err
		}
		return "", nil
	}

	if err != nil {
		if stderrBuf, ok := llmCmd.Stderr.(*bytes.Buffer); ok {
			return "", fmt.Errorf("llm-cli exited with an error: %w\nstderr: %s", err, stderrBuf.String())
		}
		return "", fmt.Errorf("llm-cli exited with an error: %w", err)
	}

	if stdoutBuf, ok := llmCmd.Stdout.(*bytes.Buffer); ok {
		return stdoutBuf.String(), nil
	}
	return "", nil
}

// worker is a goroutine that processes jobs from the jobs channel.
func worker(wg *sync.WaitGroup, jobs <-chan Job, results chan<- Result, systemPrompt, profile string) {
	defer wg.Done()
	for job := range jobs {
		output, err := processItem(job.JSONBytes, systemPrompt, profile, false)
		if err != nil {
			results <- Result{ID: job.ID, Error: err}
		} else {
			results <- Result{ID: job.ID, Input: job.ItemData, Output: output}
		}
	}
}

// peekFirstNonWhitespace peeks at the first non-whitespace character in a stream.
func peekFirstNonWhitespace(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.Peek(1)
		if err != nil {
			return 0, err
		}
		if unicode.IsSpace(rune(b[0])) {
			if _, err := r.ReadByte(); err != nil {
				return 0, err
			}
			continue
		}
		return b[0], nil
	}
}

// main is the entry point of the application.
func main() {
	log.SetFlags(0)

	// Define flags
	systemPrompt := flag.String("P", "", "The system prompt text.")
	promptFile := flag.String("F", "", "Path to a file containing the system prompt.")
	profile := flag.String("L", "", "Name of the llm-cli profile to use.")
	concurrency := flag.Int("c", 1, "Number of concurrent processes.")
	stream := flag.Bool("stream", false, "Enable stream mode. Forces -c=1, -n=1 and -o=text.")
	showVersion := flag.Bool("version", false, "Print version information and exit.")
	lines := flag.Int("n", 1, "Number of lines to batch together for each LLM call.")

	// Handle aliased flags
	var outputFormat string
	flag.StringVar(&outputFormat, "o", "text", "Output format: 'text', 'json', or 'jsonl'.")
	flag.StringVar(&outputFormat, "format", "text", "Alias for -o.")
	flag.IntVar(lines, "lines", 1, "Alias for -n.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s (-P \"<prompt>\" | -F <file>) [options] [input_file]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "A tool to process JSON array/JSONL data by sending each item/batch to an LLM.")
		fmt.Fprintln(os.Stderr, "\nArguments:")
		fmt.Fprintln(os.Stderr, "  [input_file]    Input JSON array or JSONL file. Reads from stdin if omitted.")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *stream {
		if *concurrency != 1 {
			log.Println("Warning: Stream mode enabled. Forcing concurrency to 1.")
			*concurrency = 1
		}
		if outputFormat != "text" {
			log.Println("Warning: Stream mode enabled. Forcing output format to 'text'.")
			outputFormat = "text"
		}
		if *lines != 1 {
			log.Println("Warning: Stream mode enabled. Forcing batch size (-n) to 1.")
			*lines = 1
		}
	}

	if *concurrency < 1 {
		log.Fatal("Error: Concurrency (-c) must be at least 1.")
	}
	if *lines < 1 {
		log.Fatal("Error: Batch size (-n) must be at least 1.")
	}
	if outputFormat != "text" && outputFormat != "json" && outputFormat != "jsonl" {
		log.Fatal("Error: Invalid output format. Must be 'text', 'json', or 'jsonl'.")
	}

	var systemPromptContent string
	if (*systemPrompt != "" && *promptFile != "") || (*systemPrompt == "" && *promptFile == "") {
		log.Fatal("Error: A system prompt is required. Use either -P or -F to provide one, but not both.")
	}
	if *systemPrompt != "" {
		systemPromptContent = *systemPrompt
	} else {
		content, err := os.ReadFile(*promptFile)
		if err != nil {
			log.Fatalf("Error reading prompt file %s: %v", *promptFile, err)
		}
		systemPromptContent = string(content)
	}

	var input io.Reader = os.Stdin
	inputName := "stdin"
	if flag.NArg() > 0 {
		filePath := flag.Arg(0)
		inputName = filePath
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("Error opening file %s: %v", filePath, err)
		}
		defer file.Close()
		input = file
	}

	bufferedReader := bufio.NewReader(input)
	firstByte, err := peekFirstNonWhitespace(bufferedReader)
	if err != nil && err != io.EOF {
		log.Fatalf("Error reading from input %s: %v", inputName, err)
	}

	if *stream {
		handleStream(bufferedReader, firstByte, inputName, systemPromptContent, *profile)
		return
	}

	handleConcurrent(bufferedReader, firstByte, inputName, systemPromptContent, *profile, outputFormat, *concurrency, *lines)
}

// handleStream processes items sequentially and prints output directly.
// Note: Batching (-n) is forced to 1 in stream mode.
func handleStream(reader *bufio.Reader, firstByte byte, inputName, systemPrompt, profile string) {
	log.Printf("Stream mode enabled. Processing items from %s sequentially...", inputName)
	itemCount := 0
	processFunc := func(jsonBytes []byte) {
		itemCount++
		if itemCount > 1 {
			fmt.Println("\n---")
		}
		log.Printf("--- Processing item %d ---", itemCount)
		if _, err := processItem(jsonBytes, systemPrompt, profile, true); err != nil {
			log.Printf("Error processing item %d: %v", itemCount, err)
		}
	}

	if firstByte == '[' {
		decoder := json.NewDecoder(reader)
		decoder.Token() // consume '['
		for decoder.More() {
			var item interface{}
			decoder.Decode(&item)
			jsonBytes, _ := json.Marshal(item)
			processFunc(jsonBytes)
		}
	} else {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			if len(bytes.TrimSpace(lineBytes)) == 0 {
				continue
			}
			processFunc(lineBytes)
		}
	}
	log.Printf("\nFinished processing %d item(s).", itemCount)
}

// handleConcurrent processes items using a worker pool.
func handleConcurrent(reader *bufio.Reader, firstByte byte, inputName, systemPrompt, profile, format string, concurrency, batchSize int) {
	jobs := make(chan Job, concurrency)
	results := make(chan Result, 100)
	var wg sync.WaitGroup

	for w := 1; w <= concurrency; w++ {
		wg.Add(1)
		go worker(&wg, jobs, results, systemPrompt, profile)
	}

	jobCount := 0
	go func() {
		defer close(jobs)
		sendBatch := func(batch []interface{}) {
			if len(batch) == 0 {
				return
			}
			jobCount++
			// For batching, we wrap the items in a JSON array.
			jsonBytes, err := json.Marshal(batch)
			if err != nil {
				log.Printf("Error marshalling batch %d: %v. Skipping.", jobCount, err)
				return
			}
			jobs <- Job{ID: jobCount, JSONBytes: jsonBytes, ItemData: batch}
		}

		var batch []interface{}
		if firstByte == '[' {
			log.Printf("Detected JSON array format in %s. Processing in batches of %d...", inputName, batchSize)
			decoder := json.NewDecoder(reader)
			decoder.Token() // consume '['
			for decoder.More() {
				var item interface{}
				if err := decoder.Decode(&item); err != nil {
					log.Printf("Error decoding JSON array item: %v. Skipping.", err)
					continue
				}
				batch = append(batch, item)
				if len(batch) >= batchSize {
					sendBatch(batch)
					batch = nil // Reset batch
				}
			}
		} else {
			log.Printf("Assuming JSONL format for %s. Processing in batches of %d...", inputName, batchSize)
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				lineBytes := scanner.Bytes()
				if len(bytes.TrimSpace(lineBytes)) == 0 {
					continue
				}
				var item interface{}
				if err := json.Unmarshal(lineBytes, &item); err != nil {
					log.Printf("Error decoding JSONL line: %v. Skipping line.", err)
					continue
				}
				batch = append(batch, item)
				if len(batch) >= batchSize {
					sendBatch(batch)
					batch = nil // Reset batch
				}
			}
		}
		sendBatch(batch) // Send any remaining items
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	processedResults := make(map[int]Result)
	totalItemsProcessed := 0
	for res := range results {
		if res.Error != nil {
			log.Printf("Error processing batch %d: %v. Skipping.", res.ID, res.Error)
		} else {
			processedResults[res.ID] = res
			totalItemsProcessed += len(res.Input)
		}
	}

	jsonEncoder := json.NewEncoder(os.Stdout)
	if format == "json" {
		var finalResults []Result
		for i := 1; i <= jobCount; i++ {
			if res, ok := processedResults[i]; ok {
				finalResults = append(finalResults, res)
			}
		}
		jsonEncoder.Encode(finalResults)
	} else {
		for i := 1; i <= jobCount; i++ {
			res, ok := processedResults[i]
			if !ok {
				continue
			}
			switch format {
			case "text":
				if i > 1 {
					fmt.Println("\n---")
				}
				fmt.Print(res.Output)
			case "jsonl":
				jsonEncoder.Encode(res)
			}
		}
	}

	if jobCount == 0 {
		log.Printf("Warning: No JSON items were processed from %s.", inputName)
	} else {
		log.Printf("\nSuccessfully processed %d item(s) in %d batch(es).", totalItemsProcessed, len(processedResults))
	}
}
