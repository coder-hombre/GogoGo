# GogoGo

Simple Go API with an in-memory item list and a SerpAPI-backed random-item search endpoint.
SerpAPI repository link [here](https://github.com/serpapi/serpapi-golang)

## Quick start

1. Copy the example env file:

   ```powershell
   Copy-Item .env.example .env
   ```

2. Edit `.env` and set your real key:

   ```dotenv
   SERPAPI_API_KEY=your_real_key_here
   ```

3. Run the server:

   ```powershell
   go run .
   ```

## Endpoints

- `GET /health`
- `GET /items`
- `POST /items`
- `GET /findRandomItemFromList`

## Verify

With the server running:

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/findRandomItemFromList
```

If `SERPAPI_API_KEY` is missing, `GET /findRandomItemFromList` returns a `500` error.
