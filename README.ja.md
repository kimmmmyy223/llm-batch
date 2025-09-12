# LLM Batch Processing Tool (`llm-batch`)

`llm-batch` は、JSON配列またはJSONL（JSON Lines）形式のデータを受け取り、各JSONオブジェクトをLLMに送信して一括処理を行うためのコマンドラインツールです。

デフォルトでは安全な**逐次処理**を行いますが、バックエンドが対応している場合は**並行処理**に切り替えてパフォーマンスを向上させることができます。また、**ストリーミングモード**を使用することで、`llm-cli`の出力をリアルタイムでモニタリングすることも可能です。

## 必要条件

  * **Go**: 1.18以上 (Goプログラムをビルドするため)
  * **make**: (Makefileを使用してビルドするため)
  * **llm-cli**: LLMと対話するためのクライアントツール。PATHが通っている必要があります。
  * **git**: (推奨) バージョン情報をバイナリに埋め込むために使用されます。

## ビルド方法

1.  ターミナルでこのリポジトリの `llm-batch` ディレクトリに移動します。

2.  Goモジュールを初期化します。（初回のみ必要）

    ```bash
    go mod init llm-batch
    ```

3.  以下のコマンドを実行してビルドします。

    ```bash
    make build
    ```

4.  ビルドが完了すると、`bin/` ディレクトリ内に各OS・アーキテクチャ向けの実行可能ファイルが生成されます。

      * `llm-batch-linux-amd64`
      * `llm-batch-darwin-universal` (macOS Universal Binary for Intel & Apple Silicon)
      * `llm-batch-windows-amd64.exe`

## 使い方

### コマンド構文

```bash
./<binary_name> (-P "<prompt>" | -F <file>) [options] [input_file]
```

  * `<binary_name>`: あなたの環境に合った実行可能ファイル名。（例: `bin/llm-batch-darwin-universal`）

### 引数とオプション

  * `[input_file]`: 入力となるJSON配列またはJSONLファイルへのパス。省略した場合、標準入力からデータを読み取ります。

  * **プロンプト (必須)**

      * `-P "<prompt>"`: LLMに与えるシステムプロンプトを直接指定します。
      * `-F <prompt_file>`: システムプロンプトが記述されたファイルへのパスを指定します。

  * **オプション**

      * `-L <profile>`: 使用する `llm-cli` のプロファイル名を指定します。
      * `-o, --format <format>`: 出力形式を指定します。デフォルトは `text` です。
          * `text`: LLMの出力をそのまま表示します。アイテム間は `---` で区切られます。
          * `json`: 全ての結果を一つのJSON配列として出力します。（入力順）
          * `jsonl`: 各結果を一行のJSONオブジェクトとして出力します。（入力順）
      * `-c <num>`: 同時に実行するプロセスの数を指定します。デフォルトは `1` (逐次処理) です。
      * `--stream`: `llm-cli`の出力をリアルタイムで表示するストリーミングモードを有効にします。デバッグに便利です。このオプションを使用すると、並行処理は無効化 (`-c=1`) され、出力形式は `text` に強制されます。
      * `--version`: ツールのバージョン情報を表示して終了します。

### 並行処理について (重要)

`-c` オプションで `2` 以上の値を設定すると、複数の `llm-cli` プロセスが同時に実行されます。

  * **推奨されるケース**: Amazon BedrockやGoogle Vertex AIなど、**並列リクエストに対応したクラウドAPI**を `llm-cli` のバックエンドとして使用している場合。
  * **非推奨のケース**: OllamaやLM Studioなど、**ローカルで単一のLLMを動かしている**場合。これらのモデルは一度に一つのリクエストしか処理できないことが多く、並列で呼び出すとリソースの競合を引き起こし、かえって処理が遅くなったり、システムが不安定になったりする可能性があります。ローカルモデルを使用する場合は、`-c 1` (デフォルト) のままにすることを強く推奨します。

### 実行例

#### 1\. 逐次処理 (デフォルト)

```bash
cat reviews.jsonl | ./bin/llm-batch-darwin-universal \
  -P "レビューを英語に翻訳してください。" \
  --format jsonl
```

#### 2\. ストリーミングモード (デバッグ用)

```bash
cat reviews.jsonl | ./bin/llm-batch-darwin-universal \
  -P "このレビューの問題点を指摘してください。" \
  --stream
```

#### 3\. 並行処理 (クラウドAPI向け)

Amazon Bedrockのプロファイル `my-bedrock-profile` を使い、4並列で処理する例です。

```bash
./bin/llm-batch-darwin-universal \
  -F prompt.txt \
  -L my-bedrock-profile \
  --format jsonl \
  -c 4 \
  reviews.jsonl
```