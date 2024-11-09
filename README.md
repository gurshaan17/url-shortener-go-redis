# URL Shortener Service

A URL shortening service built with Go, utilizing Redis for data storage. This project allows users to shorten URLs and supports custom short URLs with expiration settings. With Docker support, it’s easy to set up and run locally with minimal configuration.

## Features

- Shorten URLs with custom aliases.
- URL expiration settings.
- Simple rate limiting.
- Persistent data storage with Redis.
- Easily deployable with Docker and Docker Compose.

## Getting Started

### Prerequisites

- Docker and Docker Compose installed.
- Redis (optional if running the complete setup with Docker Compose).

### Clone and Run the Project

1. **Clone the repository:**

   ```bash
   git clone https://github.com/your-username/url-shortener-go-redis.git
   cd url-shortener-go-redis
   ```

2. **Run with Docker Compose:**

   ```bash
   docker-compose up -d
   ```

   This will start the application and Redis in detached mode.

### Using Docker Hub Image

Alternatively, you can pull the Docker image directly from Docker Hub:

1. **Pull the Image:**

   ```bash
   docker pull gurshaan17/url-shortener-redis-api:latest
   ```

2. **Run Redis Locally:**

   If Redis isn’t running in Docker Compose, you’ll need to run a local Redis instance:

   ```bash
   docker run --name redis -d -p 6379:6379 redis
   ```

3. **Run the Application:**

   ```bash
   docker run -d -p 8080:8080 --link redis:redis your-docker-username/url-shortener-redis-api:latest
   ```

## API Endpoints

### Shorten URL

- **Endpoint:** `/shorten`
- **Method:** `POST`
- **Request Body:**

   ```json
   {
     "url": "https://example.com",
     "short": "custom-alias", // Optional
     "expiry": 24 // Expiry time in hours, optional
   }
   ```

- **Response:**

   ```json
   {
     "url": "https://example.com",
     "short": "http://localhost:8080/custom-alias",
     "expiry": 24,
     "rate_limit": 10,
     "rate_limit_reset": 30
   }
   ```

### Rate Limiting

Each IP is allowed 10 requests every 30 minutes. If the limit is exceeded, a rate limit exceeded error is returned with the remaining cooldown.

## Environment Variables

To customize the application, you can set the following environment variables:

- `API_QUOTA`: The maximum number of requests allowed per IP (default is 10).
- `DOMAIN`: The domain used for generating short URLs.

