package main

import (
	"fmt"
	"html"
	"net/http"
	"time"
)

var quotes = []string{
	"The only way to do great work is to love what you do. - Steve Jobs",
	"Innovation distinguishes between a leader and a follower. - Steve Jobs",
	"Life is what happens to you while you're busy making other plans. - John Lennon",
	"The future belongs to those who believe in the beauty of their dreams. - Eleanor Roosevelt",
	"It is during our darkest moments that we must focus to see the light. - Aristotle",
	"Quality is not an act, it is a habit. - Aristotle",
	"The only impossible journey is the one you never begin. - Tony Robbins",
	"In the middle of difficulty lies opportunity. - Albert Einstein",
	"Success is not final, failure is not fatal: it is the courage to continue that counts. - Winston Churchill",
	"The way to get started is to quit talking and begin doing. - Walt Disney",
	"Don't let yesterday take up too much of today. - Will Rogers",
	"You learn more from failure than from success. Don't let it stop you. Failure builds character. - Unknown",
	"If you are working on something exciting that you really care about, you don't have to be pushed. The vision pulls you. - Steve Jobs",
	"People who are crazy enough to think they can change the world, are the ones who do. - Rob Siltanen",
	"We may encounter many defeats but we must not be defeated. - Maya Angelou",
	"The only person you are destined to become is the person you decide to be. - Ralph Waldo Emerson",
	"Go confidently in the direction of your dreams. Live the life you have imagined. - Henry David Thoreau",
	"The two most important days in your life are the day you are born and the day you find out why. - Mark Twain",
	"Whatever you can do, or dream you can, begin it. Boldness has genius, power and magic in it. - Johann Wolfgang von Goethe",
	"The best time to plant a tree was 20 years ago. The second best time is now. - Chinese Proverb",
	"Your limitation—it's only your imagination.",
	"Push yourself, because no one else is going to do it for you.",
	"Great things never come from comfort zones.",
	"Dream it. Wish it. Do it.",
	"Success doesn't just find you. You have to go out and get it.",
	"The harder you work for something, the greater you'll feel when you achieve it.",
	"Dream bigger. Do bigger.",
	"Don't stop when you're tired. Stop when you're done.",
	"Wake up with determination. Go to bed with satisfaction.",
	"Do something today that your future self will thank you for.",
	"Little things make big things happen.",
	"It's going to be hard, but hard does not mean impossible.",
	"Don't wait for opportunity. Create it.",
	"Sometimes we're tested not to show our weaknesses, but to discover our strengths.",
	"The key to success is to focus on goals, not obstacles.",
	"Dream it. Believe it. Build it.",
}

func getQuoteOfTheDay() string {
	// Use day of year to select a quote (ensures same quote for entire day)
	now := time.Now()
	dayOfYear := now.YearDay()
	quoteIndex := dayOfYear % len(quotes)
	return quotes[quoteIndex]
}

func quoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	quote := getQuoteOfTheDay()
	date := time.Now().Format("January 2, 2006")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Quote of the Day</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: #333;
        }
        .container {
            background: white;
            padding: 3rem;
            border-radius: 1rem;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 600px;
            text-align: center;
        }
        h1 {
            margin-top: 0;
            color: #667eea;
            font-size: 2rem;
        }
        .quote {
            font-size: 1.5rem;
            line-height: 1.6;
            margin: 2rem 0;
            color: #2d3748;
            font-style: italic;
        }
        .date {
            color: #718096;
            font-size: 1rem;
            margin-top: 2rem;
        }
        .footer {
            margin-top: 2rem;
            padding-top: 1rem;
            border-top: 1px solid #e2e8f0;
            color: #a0aec0;
            font-size: 0.875rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Quote of the Day</h1>
        <div class="quote">%s</div>
        <div class="date">%s</div>
        <div class="footer">Jump Server</div>
    </div>
</body>
</html>`, html.EscapeString(quote), html.EscapeString(date))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

func main() {
	http.HandleFunc("/", quoteHandler)

	port := ":10000"
	fmt.Printf("Quote server starting on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
