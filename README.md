# LLM Batch Processing Tool (`llm-batch`)

`llm-batch` is a command-line tool for batch processing data in JSON array or JSONL (JSON Lines) format by sending each JSON object to an LLM.

By default, it performs safe **sequential processing**, but you can switch to **parallel processing** to improve performance if your backend supports it. You can also use **streaming mode** to monitor the output of `llm-cli` in real-time.

## Requirements

*   **Go**: 1.18 or higher (to build the Go program)
*   **make**: (to build using the Makefile)
*   **llm-cli**: A client tool for interacting with LLMs. It must be in your PATH.
*   **git**: (Recommended) Used to embed version information into the binary.

## How to Build

Simply run the following command in the project root directory. It will handle dependencies and create the executables.

```bash
make build
```

Once the build is complete, executable files for each OS and architecture will be generated in the `bin/` directory.

    *   `llm-batch-linux-amd64`
    *   `llm-batch-darwin-universal` (macOS Universal Binary for Intel & Apple Silicon)
    *   `llm-batch-windows-amd64.exe`

## Usage

### Command Syntax

```bash
./<binary_name> (-P "<prompt>" | -F <file>) [options] [input_file]
```

*   `<binary_name>`: The name of the executable file for your environment (e.g., `bin/llm-batch-darwin-universal`).

### Arguments and Options

*   `[input_file]`: Path to the input JSON array or JSONL file. If omitted, data will be read from standard input.

*   **Prompt (Required)**

    *   `-P "<prompt>"`: Directly specify the system prompt to give to the LLM.
    *   `-F <prompt_file>`: Specify the path to a file containing the system prompt.

*   **Options**

    *   `-L <profile>`: Specify the `llm-cli` profile name to use.
    *   `-o, --format <format>`: Specify the output format. The default is `text`.
        *   `text`: Displays the raw output from the LLM. Items are separated by `---`.
        *   `json`: Outputs all results as a single JSON array (in input order).
        *   `jsonl`: Outputs each result as a single-line JSON object (in input order).
    *   `-c <num>`: Specify the number of concurrent processes. The default is `1` (sequential processing).
    *   `--stream`: Enable streaming mode to display `llm-cli` output in real-time. Useful for debugging. When this option is used, parallel processing is disabled (`-c=1`) and the output format is forced to `text`.
    *   `--version`: Display the tool's version information and exit.

### Regarding Parallel Processing (Important)

Setting a value of `2` or higher for the `-c` option will run multiple `llm-cli` processes simultaneously.

*   **Recommended Use Case**: When using a cloud API that supports parallel requests as the backend for `llm-cli`, such as Amazon Bedrock or Google Vertex AI.
*   **Not Recommended Use Case**: When running a single local LLM, such as with Ollama or LM Studio. These models can often only handle one request at a time, and calling them in parallel can cause resource contention, leading to slower processing or system instability. It is strongly recommended to keep the setting at `-c 1` (the default) when using local models.

### Execution Examples

#### 1. Sequential Processing (Default)

```bash
cat reviews.jsonl | ./bin/llm-batch-darwin-universal \
  -P "Translate this review into English."
  --format jsonl
```

#### 2. Streaming Mode (for Debugging)

```bash
cat reviews.jsonl | ./bin/llm-batch-darwin-universal \
  -P "Point out the problems in this review."
  --stream
```

#### 3. Parallel Processing (for Cloud APIs)

This example processes `reviews.jsonl` with 4-way parallelism using the Amazon Bedrock profile `my-bedrock-profile`.

```bash
./bin/llm-batch-darwin-universal \
  -F prompt.txt \
  -L my-bedrock-profile \
  --format jsonl \
  -c 4 \
  reviews.jsonl
```
