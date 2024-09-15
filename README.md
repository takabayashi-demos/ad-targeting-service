# ad-targeting-service

Ad targeting and personalization engine

## Tech Stack
- **Language**: go
- **Team**: marketing
- **Platform**: Walmart Global K8s

## Quick Start
```bash
docker build -t ad-targeting-service:latest .
docker run -p 8080:8080 ad-targeting-service:latest
curl http://localhost:8080/health
```

## API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| GET | /ready | Readiness probe |
| GET | /metrics | Prometheus metrics |
