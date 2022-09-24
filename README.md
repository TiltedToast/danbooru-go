A simple Go client for the Danbooru API to download images in a batch. Currently maxed out at 2 tags per search, however if you are Danbooru Gold member then this limit gets increased to 6. 

To take advantage of that, rename the `.env.example` file to `.env` and enter the required credentials. 

Just make sure you keep said `.env` file in the same directory as the executable.

This tool tends to not do so well when you're trying to download an extreme amount of images (97k+), as at some point it always starts to receive HTTP 500 responses from the server. This can be partially mitigated by increasing the rate limit applied when fetching all posts.

I found that even when fetching 1 page per second it will still error out well before hitting the end. This is obviously much better than having it error out before even hitting more than a couple hundred pages but you also have to deal with a cripplingly slow process of fetching posts.
