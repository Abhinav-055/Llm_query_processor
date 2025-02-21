# LLM Query Processor with Redis Pub/Sub and Persistent Storage

This project is a concurrent LLM query processor built in Go. It integrates with Redis to provide both real-time notifications (using Pub/Sub) and reliable result retrieval (using persistent storage). The application processes user queries asynchronously and allows you to retrieve the results via HTTP endpoints.

## Prerequisites

- [Go](https://golang.org/dl/) (version 1.16 or later)
- [Docker Desktop](https://www.docker.com/products/docker-desktop) (or Docker Engine installed locally)
- A valid OpenAI API key

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/Abhinav-055/Llm_query_processor
cd llm-query-processor
```

### 2. Set Up the Environment

Create a `.env` file in the root directory of the project with the following content:

```env
OPENAI_API_KEY=your-api-key-here
```

Replace `your-api-key-here` with your actual OpenAI API key.

### 3. Run Redis with Docker

Start a Redis container using Docker. You can use Docker Desktop or run the following command in your terminal:

```bash
docker run -d --name redis -p 6379:6379 redis:latest
```

This command runs Redis in detached mode and maps port `6379` of the container to port `6379` on your host machine.

### 4. Run the Application

In the project directory, run:

```bash
go run .
```

This will start the server on port `8080`.

## Using the API

### **POST /query**

Send a `POST` request to `http://localhost:8080/query` with a JSON payload containing a single prompt. For example:

```json
{
  "prompt": "Hello, how can I assist you today?"
}
```

The server will respond with a JSON object that includes a unique query ID, for example:

```json
{
  "message": "Query received",
  "id": 1
}
```

### **GET /result**

Send a `GET` request to retrieve the result of a query using its ID:

```bash
http://localhost:8080/result?id=1
```

The server will respond with the processed result in JSON format. If you configured your response to omit an ID when zero, you might see:

```json
{
  "text": "Hello! How can I assist you today?"
}
```

## How It Works

### **POST Handler (`/query`)**:

- Accepts a JSON payload with a single prompt.
- Assigns a unique ID to the query.
- Enqueues the query on a channel for worker processing.
- Returns a response with the query ID.

### **Worker Processing**:

- Workers pick up queries from the channel and process them by calling the LLM (via OpenAI API).
- After processing, each worker publishes the result to a Redis Pub/Sub channel (`query_result_<id>`) and also stores the result persistently in Redis using a key (the same channel name).

### **GET Handler (`/result`)**:

- When a `GET` request is made with a query ID, the handler first checks Redis for a stored result.
- If a stored result is found, it’s returned immediately.
- Otherwise, the handler subscribes to the Redis Pub/Sub channel to wait for a live update.
- The result is then unmarshaled (removing any extra escape characters) and sent as the JSON response.

## Project Structure

- `main.go`: Contains the HTTP server setup, Redis client initialization, and API handlers.
- `openai.go`: Contains functions for calling the LLM (OpenAI API).
- `worker.go`: Contains the worker pool that processes queries and publishes results to Redis.

## License

This project is licensed under the MIT License.

