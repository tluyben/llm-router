# 🚀 OpenAI and Anthropic to OpenRouter API Router

## 🌟 Overview

This project provides a Golang-based API router that intercepts calls to both the OpenAI and Anthropic APIs and redirects them to OpenRouter. It's designed to be a drop-in replacement for OpenAI and Anthropic API clients, allowing you to use OpenRouter's services transparently.

## ✨ Features

- 🔄 Seamless routing of OpenAI and Anthropic API calls to OpenRouter
- 🌊 Support for streaming responses
- 🧠 Optional JavaScript middleware for request processing
- 🛣️ Custom routing logic using JavaScript
- 💬 Optional system prompt injection
- 🐳 Docker support for easy deployment

## 🛠️ Prerequisites

- Go 1.16 or higher
- Docker (optional, for containerized deployment)
- xDocker (https://github.com/tluyben/xdocker)

## 🚀 Getting Started

### 🔧 Installation

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/openai-anthropic-openrouter-router.git
   cd openai-anthropic-openrouter-router
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

### ⚙️ Configuration

1. Create a `.env` file in the project root with the following contents:

   ```
   OR_MODEL=your_openrouter_model
   OR_KEY=your_openrouter_api_key
   OR_ENDPOINT=https://openrouter.ai/api/v1/chat/completions
   ```

2. (Optional) Create a `middleware.js` file if you want to process requests:

   ```javascript
   function process(request) {
     // Modify the request object here
     return request;
   }
   ```

3. (Optional) Create a `system_prompt.txt` file if you want to inject a system prompt:
   ```
   You are a helpful assistant.
   ```

### 🏃‍♂️ Running the Application

#### 🖥️ Locally

Run the application with the following command:

```
go run main.go --middleware middleware.js --system system_prompt.txt
```

#### 🐳 Docker Setup

##### 🏗️ Building the Docker Image

To build the Docker image for this project, run the following command in the project root directory:

```bash
docker build -t openai-anthropic-openrouter-router .
```

##### 🏃‍♂️ Running with Docker

To run the container with default settings:

```bash
docker run -p 80:80 openai-anthropic-openrouter-router
```

To include optional components (middleware, system prompt, or custom router), you can mount them as volumes:

```bash
docker run -p 80:80 \
  -v $(pwd)/.env:/root/.env \
  -v $(pwd)/middleware.js:/root/middleware.js \
  -v $(pwd)/system_prompt.txt:/root/system_prompt.txt \
  -v $(pwd)/router.js:/root/router.js \
  openai-anthropic-openrouter-router \
  ./router --nohosts --middleware /root/middleware.js --system /root/system_prompt.txt --router /root/router.js
```

Adjust the command line arguments based on which optional components you want to include.

##### 🐙 Using Docker Compose

For easier management, you can use Docker Compose. Create a `docker-compose.yml` file in your project root with the following content:

```yaml
services:
  api-router:
    build: .
    ports:
      - "80:80"
    volumes:
      - ./.env:/root/.env
      # Uncomment the following lines if you want to include optional components
      # - ./middleware.js:/root/middleware.js
      # - ./system_prompt.txt:/root/system_prompt.txt
      # - ./router.js:/root/router.js
    environment:
      - OR_MODEL=${OR_MODEL}
      - OR_KEY=${OR_KEY}
      - OR_ENDPOINT=${OR_ENDPOINT}
    command: ["./router", "--nohosts"]
    # Uncomment and adjust the following line to include optional components
    # command: ["./router", "--nohosts", "--middleware", "/root/middleware.js", "--system", "/root/system_prompt.txt", "--router", "/root/router.js"]
```

To start the service, run:

```bash
docker-compose up --build
```

To stop the service, use:

```bash
docker-compose down
```

Remember to create and configure your `.env`, `middleware.js`, `system_prompt.txt`, and `router.js` files as needed before running the Docker container or Docker Compose.

## 🎯 Usage

After starting the router (either directly or via Docker), it will listen on port 80. To use it:

1. If running directly on your machine, the router will offer to modify your `/etc/hosts` file to point both `api.openai.com` and `api.anthropic.com` to `127.0.0.1`.

2. If using Docker, you'll need to manually modify your host machine's `/etc/hosts` file to add these entries:

   ```
   127.0.0.1 api.openai.com
   127.0.0.1 api.anthropic.com
   ```

3. Use your OpenAI or Anthropic API client as usual, it will now be routed through OpenRouter.

4. For streaming responses, include `?stream=true` in your API requests.

## 🧪 Examples

### OpenAI API (Chat Completion)

```python
import openai

openai.api_key = "your-openai-api-key"  # This won't be used, but is required by the client
openai.api_base = "http://localhost/v1"  # Point to your local router

response = openai.ChatCompletion.create(
    model="gpt-3.5-turbo",  # This will be overridden with the OR_MODEL
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ]
)

print(response.choices[0].message.content)
```

### Anthropic API

```python
import anthropic

client = anthropic.Client(api_key="your-anthropic-api-key")  # This won't be used, but is required by the client
client.base_url = "http://localhost/v1"  # Point to your local router

response = client.completion(
    model="claude-2",  # This will be overridden with the OR_MODEL
    prompt="Human: Hello, how are you?\n\nAssistant:",
    max_tokens_to_sample=300,
)

print(response.completion)
```

### Streaming Response (OpenAI style)

```python
import openai

openai.api_key = "your-openai-api-key"
openai.api_base = "http://localhost/v1"

for chunk in openai.ChatCompletion.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Tell me a story about a robot."}
    ],
    stream=True
):
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- OpenAI and Anthropic for their API designs
- OpenRouter for providing alternative AI model access
- The Go community for the excellent libraries used in this project
