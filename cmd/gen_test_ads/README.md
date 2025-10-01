# Ad Generator

This tool generates realistic automotive parts advertisements using the Grok API. It randomly selects vehicle makes, years, models, and engines from the available data and uses AI to generate authentic ad content.

## Usage

```bash
# Generate 30 ads (default) and output to stdout
go run ./cmd/gen_ads

# Generate 50 ads and save to file
go run ./cmd/gen_ads -count 50 -output new_ads.json

# Generate ads with specific user ID and start date
go run ./cmd/gen_ads -count 20 -user-id 2 -start-date 2024-06-01

# Use a specific random seed for reproducible results
go run ./cmd/gen_ads -count 10 -seed 12345

# Generate ads with parallel processing (faster)
go run ./cmd/gen_ads -count 50 -workers 8

# Generate ads with debug output (shows Grok API details)
go run ./cmd/gen_ads -count 5 -debug
```

## Command Line Options

- `-count`: Number of ads to generate (default: 30)
- `-output`: Output file path (default: stdout)
- `-seed`: Random seed for reproducible results (default: current timestamp)
- `-user-id`: User ID for generated ads (default: 1)
- `-start-date`: Start date for created_at timestamps in YYYY-MM-DD format (default: 2024-01-01)
- `-workers`: Number of parallel workers (default: 8, max: 8 to respect rate limits)
- `-debug`: Enable debug output (shows Grok API requests/responses) (default: false)

## Requirements

- `GROK_API_KEY` environment variable must be set
- The tool reads from `../../cmd/rebuild_db/make-year-model.json` and `../../cmd/rebuild_db/part.json`

## Output Format

The generated ads match the structure of `cmd/rebuild_db/ad.json`:

```json
[
  {
    "make": "BMW",
    "years": ["1955", "1956"],
    "models": ["501", "502"],
    "engines": ["2.0L L6", "2.6L V8"],
    "title": "BMW 501/502 Engine Oil Filter Housing",
    "description": "Original BMW oil filter housing for 1955-1956 501 and 502 models...",
    "price": 125.00,
    "created_at": "2024-01-15T10:30:00Z",
    "user_id": 2,
    "category": "Engine",
    "subcategory": "",
    "location": {
      "city": "Munich",
      "admin_area": "Bavaria",
      "country": "DE",
      "latitude": 48.1351,
      "longitude": 11.5820
    }
  }
]
```

## Features

- **Parallel Processing**: Uses multiple workers to generate ads concurrently
- **Rate Limiting**: Respects Grok API limits (480 requests/minute) with built-in throttling
- **Weighted Make Selection**: Favors popular American makes (Ford, Chevrolet, Dodge) while including foreign and lesser-known makes
- **Random Selection**: Selects 1-3 years, 1-3 models, and all compatible engines
- **Validation**: Ensures all combinations have non-empty make, years, models, and engines (retries up to 10 times)
- **AI-Generated Content**: Uses Grok AI to generate realistic titles, descriptions, and prices
- **Concise Titles**: Creates titles with make and year range (no engine details)
- **Focused Descriptions**: Generates descriptions without repeating vehicle details
- **Smart Locations**: Selects appropriate locations based on vehicle make
- **Random Timestamps**: Generates timestamps within the specified date range
- **Random User IDs**: Assigns random user IDs (1-4)

## Make Selection Weights

The tool uses weighted selection to favor popular American makes while still including foreign and lesser-known makes:

### Popular American Makes (Very High Weight)
- **Ford**: 200 (most popular)
- **Chevrolet**: 180
- **Dodge**: 160
- **Chrysler**: 140
- **Buick, Cadillac, GMC, Jeep**: 120 each
- **Lincoln, Mercury, Pontiac**: 100 each
- **Oldsmobile, Plymouth, RAM**: 80 each
- **Hummer, Saturn, Tesla**: 60 each

### Popular Foreign Makes (Low Weight)
- **BMW, Audi, Mercedes, Volkswagen**: 8 each
- **Toyota, Honda, Nissan**: 8 each
- **Mazda, Subaru, Infiniti, Acura, Lexus**: 5 each
- **Porsche**: 3
- **Ferrari, Lamborghini, Maserati, Jaguar, Bentley, Rolls, Aston, McLaren, Lotus, Mini, Land Rover, Alfa Romeo, Fiat, Abarth**: 2 each

### Lesser-Known Makes (Low Weight)
- All other makes: 1-3 (random weight)

### Expected Distribution

- **~85%**: Popular American makes (Ford, Chevrolet, Dodge, etc.)
- **~12%**: Popular foreign makes (BMW, Audi, Toyota, etc.)
- **~3%**: Lesser-known makes (Sterling, Denzel, etc.)
